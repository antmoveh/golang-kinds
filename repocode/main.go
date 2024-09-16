package main

import (
	"errors"
	"github.com/spf13/cobra"
	"time"
)

var options Options

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "repo",
	Short: "sync gitlab projects",
	Long:  `sync gitlab projects`,
	Example: `
repo.exe -url <url> -t <token> -o <output>
`,
	// Uncomment the following line if your bare application
	// 参数检查
	Args: func(cmd *cobra.Command, args []string) error {
		if options.Url == "" {
			return errors.New("url required")
		}
		if options.Token == "" {
			return errors.New("token required")
		}
		if options.Output == "" {
			return errors.New("output required")
		}
		return nil
	},
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		options.Args = args
		options.sync()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func main() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize()

	rootCmd.PersistentFlags().StringVar(&options.Url, "url", "", "gitlab url")
	rootCmd.PersistentFlags().StringVar(&options.Token, "token", "", "gitlab token")
	rootCmd.PersistentFlags().StringVar(&options.Output, "output", "", "output dir")
	rootCmd.PersistentFlags().IntVar(&options.Number, "number", 5, "number of projects")
	rootCmd.PersistentFlags().DurationVar(&options.TimeOut, "timeout", 30*time.Second, "time out")
}
