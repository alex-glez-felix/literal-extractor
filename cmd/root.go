package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var (
	pageID         int
	outputFile     string
	isInteractive  bool
	sourceHTMLFile string
)

var literalRegex = regexp.MustCompile(`@PageLiteralsHelper\.GetLiteral\("([^"]+)"[^)]*\)`)

var rootCmd = &cobra.Command{
	Use:   "literal-extractor [archivo1.cshtml]...",
	Short: "Extrae literales y opcionalmente sus valores de un HTML de referencia.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var sqlBuilder strings.Builder
		interactiveScanner := bufio.NewScanner(os.Stdin)
		totalLiteralsFound := 0

		var sourceLines []string
		if sourceHTMLFile != "" {
			file, err := os.Open(sourceHTMLFile)
			if err != nil {
				return fmt.Errorf("no se pudo abrir el archivo HTML de referencia: %w", err)
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				sourceLines = append(sourceLines, scanner.Text())
			}
		}

		for _, filePath := range args {
			fmt.Printf("Procesando archivo: %s\n", filePath)
			cshtmlFile, err := os.Open(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Advertencia: no se pudo leer el archivo %s: %v\n", filePath, err)
				continue
			}
			defer cshtmlFile.Close()

			lineScanner := bufio.NewScanner(cshtmlFile)
			for lineScanner.Scan() {
				line := strings.TrimSpace(lineScanner.Text())

				matches := literalRegex.FindStringSubmatch(line)
				if len(matches) < 2 {
					continue
				}

				totalLiteralsFound++
				literalKey := matches[1]
				defaultValue := ""

				foundInSource := false
				if len(sourceLines) > 0 {
					searchPatternStr := literalRegex.ReplaceAllString(line, `\s*(.*)\s*`)

					parts := strings.Split(searchPatternStr, `\s*(.*)\s*`)
					searchPattern, err := regexp.Compile(regexp.QuoteMeta(parts[0]) + `\s*(.*)\s*` + regexp.QuoteMeta(parts[1]))

					if err == nil {
						for _, sourceLine := range sourceLines {
							trimmedSourceLine := strings.TrimSpace(sourceLine)
							sourceMatch := searchPattern.FindStringSubmatch(trimmedSourceLine)

							if len(sourceMatch) > 1 {
								defaultValue = strings.TrimSpace(sourceMatch[1])
								foundInSource = true
								break
							}
						}
					}
				}

				if !foundInSource && isInteractive {
					fmt.Printf("  > Introduce el valor para '%s': ", literalKey)
					if interactiveScanner.Scan() {
						defaultValue = interactiveScanner.Text()
					}
				}

				if len(sourceLines) > 0 && !foundInSource {
					sqlBuilder.WriteString(fmt.Sprintf("-- No se encontro default value para: '%s'\n", literalKey))
					fmt.Fprintf(os.Stderr, "Advertencia: No se encontró valor para '%s' en el archivo fuente.\n", literalKey)
				}

				escapedValue := strings.ReplaceAll(defaultValue, "'", "''")
				sqlBuilder.WriteString(fmt.Sprintf("INSERT INTO `repo_literals`.`LiteralKey` (`IdPage`, `IdApp`, `IdStatus`, `Key`, `IdBusiness`) VALUES (%d,2,2,'%s',2);\n", pageID, literalKey))
				sqlBuilder.WriteString(fmt.Sprintf("INSERT INTO `repo_literals`.`LiteralDefaultValue` (`IdLanguage`, `Value`, IdLiteralKey) VALUES (47,'%s',LAST_INSERT_ID());\n\n", escapedValue))
			}
		}

		if totalLiteralsFound == 0 {
			fmt.Println("No se encontraron literales en los archivos proporcionados.")
			return nil
		}
		err := os.WriteFile(outputFile, []byte(sqlBuilder.String()), 0644)
		if err != nil {
			return fmt.Errorf("error al escribir el archivo de salida: %w", err)
		}
		fmt.Printf("\n¡Éxito! Proceso completado. Revisa el archivo '%s'.\n", outputFile)
		return nil
	},
}

func init() {
	rootCmd.Flags().IntVarP(&pageID, "page-id", "p", 0, "El ID de la página para las sentencias SQL")
	rootCmd.MarkFlagRequired("page-id")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "generated_literals.sql", "Nombre del archivo de salida para el script SQL")
	rootCmd.Flags().BoolVarP(&isInteractive, "interactive", "i", false, "Activa el modo interactivo si no se encuentra un valor")

	rootCmd.Flags().StringVarP(&sourceHTMLFile, "source-html", "s", "", "Ruta al archivo HTML de referencia para extraer valores por defecto")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
