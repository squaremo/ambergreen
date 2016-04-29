package main

import (
	"fmt"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/weaveworks/flux/common/store"
)

type queryOpts struct {
	baseOpts
	selector

	host    string
	state   string
	rule    string
	service string
	format  string
	quiet   bool
}

func (opts *queryOpts) makeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "display instances selected by the given filter",
		Long:  "Display instances selected using the given filter. By default the results are displayed in a table.",
		RunE:  opts.run,
	}
	opts.addSelectorVars(cmd)
	cmd.Flags().StringVarP(&opts.service, "service", "s", "", "print only instances in <service>")
	cmd.Flags().StringVar(&opts.host, "host", "", "select only containers on the given host")
	cmd.Flags().StringVar(&opts.state, "state", "", `select only containers in the given state (e.g., "live")`)
	cmd.Flags().StringVar(&opts.rule, "rule", "", "show only containers selected by the rule named")
	cmd.Flags().StringVarP(&opts.format, "format", "f", "", "format each instance according to the go template given (overrides --quiet)")
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "print only instance names, one to a line")
	return cmd
}

type instanceForFormat struct {
	Service string `json:"service"`
	Name    string `json:"name"`
	State   string `json:"state"`
	store.Instance
}

const (
	tableHeaders     = "SERVICE\tINSTANCE\tADDRESS\tSTATE\t\n"
	tableRowTemplate = "{{.Service}}\t{{.Name}}\t{{.Address}}\t{{.State}}"
)

func (opts *queryOpts) run(_ *cobra.Command, args []string) error {
	sel := opts.makeSelector()

	if opts.host != "" {
		sel[store.HostLabel] = opts.host
	}
	if opts.state != "" {
		sel[store.StateLabel] = opts.state
	}
	if opts.rule != "" {
		sel[store.RuleLabel] = opts.rule
	}

	var printInstance func(svcName string, svc *store.ServiceInfo, instName string, inst store.Instance) error
	if opts.quiet {
		printInstance = func(_ string, _ *store.ServiceInfo, instName string, _ store.Instance) error {
			fmt.Fprintln(opts.getStdout(), instName)
			return nil
		}
	} else {
		out := opts.getStdout()

		var tmpl *template.Template
		if opts.format == "" {
			tout := tabwriter.NewWriter(out, 4, 0, 2, ' ', 0)
			defer tout.Flush()
			tmpl = template.Must(template.New("row").Parse(tableRowTemplate))
			out = tout
			out.Write([]byte(tableHeaders))
		} else {
			tmpl = template.Must(template.New("instance").Funcs(extraTemplateFuncs).Parse(opts.format))
		}

		printInstance = func(svcName string, svc *store.ServiceInfo, instName string, inst store.Instance) error {
			err := tmpl.Execute(out, instanceForFormat{
				Service:  svcName,
				Name:     instName,
				State:    inst.Label(store.StateLabel),
				Instance: inst,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(out)
			return nil
		}
	}

	svcs := make(map[string]*store.ServiceInfo)
	var err error
	if opts.service == "" {
		svcs, err = opts.store.GetAllServices(store.QueryServiceOptions{WithInstances: true})
	} else {
		svcs[opts.service], err = opts.store.GetService(opts.service, store.QueryServiceOptions{WithInstances: true})
	}

	if err != nil {
		return err
	}

	for svcName, svc := range svcs {
		for instName, inst := range svc.Instances {
			if !sel.Includes(&inst) {
				continue
			}

			if err := printInstance(svcName, svc, instName, inst); err != nil {
				return err
			}
		}
	}

	return nil
}
