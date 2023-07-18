package cmd

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/seanlatimer/go-vanity-url/assets"
	"github.com/seanlatimer/go-vanity-url/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	tmpl       *template.Template
	configFile string
	outputDir  string
)

var rootCmd = &cobra.Command{
	Use: "vanity-url",
	Run: run,
}

// init initializes the program.
//
// It parses the templates, initializes the configuration, and sets the values for
// the command line flags.
// No parameters.
// No return types.
func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (default is vanity.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "out", "o", "build", "output directory (default is build)")
}

// initConfig initializes the configuration for the application.
func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("vanity")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatal().Err(err).Msg("config not found")
		} else {
			log.Fatal().Err(err).Msg("failed loading config")
		}
	}

	log.Info().Str("config-file", viper.ConfigFileUsed()).Msg("config loaded")
}

// Execute runs the root command and handles any errors.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("an unknown error occurred")
	}
}

// run executes the main logic of the program.
func run(cmd *cobra.Command, args []string) {
	var err error
	config := config.Config{}
	if err = viper.Unmarshal(&config); err != nil {
		log.Fatal().Err(err).Msg("failed parsing config")
	}
	tmpl, err = template.New("").Funcs(template.FuncMap{
		"urlWithoutProtocol": func(s string) string {
			u, err := url.Parse(s)
			if err != nil {
				log.Error().Err(err).Str("url", s).Msg("failed parsing url")
			}
			joined, err := url.JoinPath(u.Host, u.Path)
			if err != nil {
				log.Error().Err(err).Str("url", s).Msg("failed joining url")
			}
			return joined
		},
	}).ParseFS(assets.Content, "templates/*.tmpl")
	if err != nil {
		log.Fatal().Err(err).Msg("failed parsing templates")
	}

	if outputDir == "" {
		outputDir = "build"
	}

	if outputDir, err = filepath.Abs(outputDir); err != nil {
		log.Fatal().Err(err).Str("output-dir", outputDir).Msg("failed resolving output directory")
	}

	if err = createOutputDir(outputDir); err != nil {
		log.Fatal().Err(err).Str("output-dir", outputDir).Msg("failed creating output directory")
	}

	index, err := execTempl("index.tmpl", config)
	if err != nil {
		log.Error().Err(err).Str("template", "index.tmpl").Msg("failed to execute template")
	}

	if err = writeFile(outputDir, "index.html", index); err != nil {
		log.Error().Err(err).Str("output-dir", outputDir).Str("file", "index.html").Msg("failed writing output file")
	}

	var data []byte
	for _, repo := range config.Repos {
		if repo.Branch == "" {
			repo.Branch = "main"
		}

		if data, err = execTempl("redir.tmpl", repo); err != nil {
			log.Error().Err(err).Str("template", "redir.tmpl").Any("repo", repo).Msg("failed to execute template")
		}
		writeFile(outputDir, repo.Path+".html", data)
	}

}

// createOutputDir deletes the existing directory at the given path `out`, if it exists, and creates a new directory with the same path.
//
// Parameters:
// - out: the path of the directory to be created.
//
// Returns:
// - an error if there was an error deleting the existing directory or creating the new directory.
func createOutputDir(out string) error {
	if err := os.RemoveAll(out); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(out, os.ModeDir)
}

// execTempl executes the template with the given name and data and returns the rendered output as a byte slice.
//
// Parameters:
// - name: The name of the template to execute.
// - data: The data to pass to the template.
//
// Return type:
// - []byte: The rendered output as a byte slice.
// - error: An error if there was a problem executing the template.
func execTempl(name string, data any) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := tmpl.ExecuteTemplate(buf, name, data)
	return buf.Bytes(), err
}

func writeFile(dir string, path string, data []byte) error {
	out, err := filepath.Abs(filepath.Join(dir, path))
	if err != nil {
		return fmt.Errorf("failed resolving path for output file: %s %w", path, err)
	}

	fileDir := filepath.Dir(out)
	if err := os.MkdirAll(fileDir, os.ModeDir); err != nil && !os.IsExist(err) {
		return err
	}

	return os.WriteFile(out, data, 0755)
}
