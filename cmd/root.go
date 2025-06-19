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
	pageID        int
	outputFile    string
	isInteractive bool
)

var rootCmd = &cobra.Command{
	Use:   "literal-extractor [archivo.cshtml]",
	Short: "Extrae llaves de literales y genera un script SQL.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("error al leer el archivo %s: %w", filePath, err)
		}
		re := regexp.MustCompile(`@PageLiteralsHelper\.GetLiteral\("([^"]+)"`)
		matches := re.FindAllStringSubmatch(string(content), -1)
		if len(matches) == 0 {
			fmt.Println("No se encontraron literales en el archivo.")
			return nil
		}

		var sqlBuilder strings.Builder
		scanner := bufio.NewScanner(os.Stdin)

		for _, match := range matches {
			if len(match) > 1 {
				literalKey := match[1]
				defaultValue := ""

				if isInteractive {
					fmt.Printf("Introduce el valor para la llave '%s': ", literalKey)
					if scanner.Scan() {
						defaultValue = scanner.Text()
					}
				}

				escapedValue := strings.ReplaceAll(defaultValue, "'", "''")

				sqlBuilder.WriteString(fmt.Sprintf("INSERT INTO `repo_literals`.`LiteralKey` (`IdPage`, `IdApp`, `IdStatus`, `Key`, `IdBusiness`) VALUES (%d,2,2,'%s',2);\n", pageID, literalKey))
				sqlBuilder.WriteString(fmt.Sprintf("INSERT INTO `repo_literals`.`LiteralDefaultValue` (`IdLanguage`, `Value`, IdLiteralKey) VALUES (47,'%s',LAST_INSERT_ID());\n\n", escapedValue))
			}
		}

		err = os.WriteFile(outputFile, []byte(sqlBuilder.String()), 0644)
		if err != nil {
			return fmt.Errorf("error al escribir el archivo de salida: %w", err)
		}
		fmt.Printf("¡Éxito! Proceso completado. Revisa el archivo '%s'.\n", outputFile)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntVarP(&pageID, "page-id", "p", 0, "El ID de la página para las sentencias SQL")
	err := rootCmd.MarkFlagRequired("page-id")
	if err != nil {
		fmt.Println(err)
	}

	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "generated_literals.sql", "Nombre del archivo de salida para el script SQL")

	rootCmd.Flags().BoolVarP(&isInteractive, "interactive", "i", false, "Activa el modo interactivo para definir valores por defecto")
}
