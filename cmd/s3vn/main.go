// Copyright © 2018 YAMASAKI Masahide <masahide.y@gmail.com>
//
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/k0kubun/pp"
	"github.com/masahide/s3vn/pkg/s3vn"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	conf s3vn.Conf
	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "s3vn",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		//	Run: func(cmd *cobra.Command, args []string) { },
	}

	commitCmd = &cobra.Command{
		Use:   "commit",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: commitCmdRun,
	}

	// initCmd represents the init command
	initCmd = &cobra.Command{
		Use:   "init <Repository name> <s3 bucket name>",
		Short: "init repository config",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run:  initCmdRun,
		Args: cobra.RangeArgs(2, 2),
	}

	// configCmd represents the init command
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "setting config",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run:  configCmdRun,
		Args: cobra.MinimumNArgs(0),
	}

	cfgDir        string
	chbucket      string
	workDirViper  *viper.Viper
	workDirCfgDir string
)

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgDir, "configDir", "", "config file Dir (default is $HOME/.s3vn/)")
	rootCmd.PersistentFlags().StringVarP(&conf.WorkDir, "workdir", "w", "./", "working dir")
	rootCmd.PersistentFlags().IntVarP(&conf.MaxWorker, "worker", "", 0, "max worker. default is the same as the number of CPU cores")
	rootCmd.PersistentFlags().IntVarP(&conf.MaxFiles, "maxfiles", "n", 10000, "max files.")
	rootCmd.PersistentFlags().BoolVarP(&conf.Force, "force", "f", false, "force mode")
	rootCmd.PersistentFlags().BoolVarP(&conf.PrintLog, "verbose", "v", false, "verbose output")

	configCmd.Flags().StringVarP(&conf.UserName, "user.name", "u", os.Getenv("USER"), "set username")
	configCmd.Flags().StringVarP(&chbucket, "chbucket", "b", "", "change s3 bucket")
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
}

func configCmdRun(cmd *cobra.Command, args []string) {
}
func commitCmdRun(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	sess := session.Must(session.NewSession())
	sn := s3vn.New(sess, conf)

	if conf.PrintLog {
		pp.Println(conf) // nolint:errcheck
	}
	sn.Commit(ctx, "./")

}

func initCmdRun(cmd *cobra.Command, args []string) {
	if err := workDirViper.ReadInConfig(); err == nil {
		log.Println("Configuration already exists")
		if !conf.Force {
			os.Exit(1)
		}
	}
	conf.RepoName = args[0]
	conf.S3bucket = args[1]
	log.Println("cfgDir:", cfgDir)
	absWorkDir, err := filepath.Abs(conf.WorkDir)
	if err != nil {
		log.Println(err)
	}
	log.Println("workDir:", absWorkDir)
	workDirViper.Set("RepoName", conf.RepoName)
	workDirViper.Set("S3bucket", conf.S3bucket)
	workDirViper.Set("WorkDir", conf.WorkDir)
	workDirViper.Set("MaxWorker", conf.MaxWorker)
	workDirViper.Set("MaxFiles", conf.MaxFiles)
	if err := workDirViper.WriteConfig(); err != nil {
		log.Fatalf("failed WriteConfig: %s", err)
	}
	pp.Println(conf) // nolint:errcheck

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	cfgFile := "config.yaml"
	//pp.Println("initConfig....")
	if cfgDir != "" {
		viper.SetConfigFile(filepath.Join(cfgDir, cfgFile))
		viper.SetConfigName("config")

	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		cfgDir = filepath.Join(home, ".s3vn")
		if _, err := os.Stat(cfgDir); err != nil {
			if err := os.Mkdir(cfgDir, 0700); err != nil {
				log.Println(err)
				os.Exit(1)
			}
		}
		viper.AddConfigPath(cfgDir)
		viper.SetConfigFile(cfgFile)
		// Use config file from the flag.
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if err := viper.WriteConfig(); err == nil {
			log.Printf("failed WriteConfig: %s", err)
		}
		log.Println("Using config file:", viper.ConfigFileUsed())
	}
	initWorkdir()
}

func initWorkdir() {
	if err := os.Chdir(conf.WorkDir); err != nil {
		log.Fatalf("failed change directory:%s", conf.WorkDir)
	}
	workPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	workDirCfgDir = filepath.Join(cfgDir, workPath)
	//pp.Println("workDirCfgDir:", workDirCfgDir)
	//pp.Println("cfgDir:", cfgDir)
	if _, err := os.Stat(workDirCfgDir); err != nil {
		if err := os.MkdirAll(workDirCfgDir, 0700); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}
	workDirViper = viper.New()
	confFile := filepath.Join(workDirCfgDir, "config.yaml")
	workDirViper.SetConfigFile(confFile)
	if err := workDirViper.ReadInConfig(); err == nil {
		conf.RepoName = workDirViper.GetString("RepoName")
		conf.S3bucket = workDirViper.GetString("S3bucket")
		conf.WorkDir = workDirViper.GetString("WorkDir")
		conf.MaxWorker = workDirViper.GetInt("MaxWorker")
		conf.MaxFiles = workDirViper.GetInt("MaxFiles")
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func main() {
	execute()
}
