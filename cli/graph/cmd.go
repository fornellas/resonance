package graph

// import (
// 	"fmt"
// 	"net/url"
// 	"strings"

// 	"github.com/spf13/cobra"

// 	"github.com/fornellas/resonance/cli/lib"
// 	"github.com/fornellas/resonance/log"
// )

// func linkFormat(dot string) {
// 	v := url.Values{}
// 	v.Add("dot", dot)
// 	fmt.Printf("http://magjac.com/graphviz-visual-editor/?%s", v.Encode())
// }

// func dotFormat(dot string) {
// 	fmt.Printf("%s", dot)
// }

// var format string
// var defaultFormat = "link"
// var formatFnMap = map[string]func(string){
// 	"link": linkFormat,
// 	"dot":  dotFormat,
// }

// var Cmd = &cobra.Command{
// 	Use:   "graph [flags] resources_root",
// 	Short: "Show a graph of the plan to apply.",
// 	Long:  "Loads all resoures from .yaml files at resources_root, the previous state, craft a plan graph and show it.",
// 	Args:  cobra.ExactArgs(1),
// 	Run: func(cmd *cobra.Command, args []string) {
// 		ctx := cmd.Context()
// 		logger := log.GetLogger(ctx)

// 		formatFn, ok := formatFnMap[format]
// 		if !ok {
// 			logger.Fatalf("invalid format: %s", format)
// 		}

// 		root := args[0]

// 		// Host
// 		hst, err := lib.GetHost(ctx)
// 		if err != nil {
// 			logger.Fatal(err)
// 		}
// 		defer hst.Close()

// 		// PersistantState
// 		persistantState, err := lib.GetPersistantState(hst)
// 		if err != nil {
// 			logger.Fatal(err)
// 		}

// 		// Plan
// 		_, plan, _ := lib.Plan(ctx, hst, persistantState, root)

// 		formatFn(plan.Graphviz())
// 	},
// }

// func Reset() {
// 	format = defaultFormat
// }

// func init() {
// 	lib.AddHostFlags(Cmd)
// 	lib.AddPersistantStateFlags(Cmd)
// 	formats := []string{}
// 	for format := range formatFnMap {
// 		formats = append(formats, format)
// 	}
// 	Cmd.Flags().StringVarP(
// 		&format, "format", "f", defaultFormat,
// 		fmt.Sprintf("Graph format: %s.", strings.Join(formats, ", ")),
// 	)
// }
