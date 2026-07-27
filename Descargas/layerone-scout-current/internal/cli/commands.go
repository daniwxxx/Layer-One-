package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"layerone-scout/internal/app"
	"layerone-scout/internal/config"
	"layerone-scout/internal/fetcher"
	"layerone-scout/internal/model"
	"layerone-scout/internal/server"
	"layerone-scout/internal/storage"
	"layerone-scout/pkg/utils"
)

func RunCLI(args []string) {
	if len(args) < 2 {
		printGlobalHelp()
		os.Exit(1)
	}
	cfg, err := config.Load("scout.yaml", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error cargando config: %v\n", err)
		os.Exit(1)
	}
	fetcher.Init(cfg.FetcherRuntime())

	cmd := args[1]
	switch cmd {
	case "ls", "list":
		cmd = "list"
	case "rm", "delete":
		cmd = "delete"
	}

	store, err := storage.NewJSONStore(cfg.Storage.Path, cfg.Storage.BackupDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error abriendo store: %v\n", err)
		os.Exit(1)
	}
	appInst := app.New(store, cfg)

	var jsonOutput bool
	var debug bool
	subArgs := args[2:]
	var filtered []string
	for _, a := range subArgs {
		if a == "--json" {
			jsonOutput = true
		} else if a == "--debug" {
			debug = true
		} else {
			filtered = append(filtered, a)
		}
	}
	_ = debug

	switch cmd {
	case "init":
		handleInit(store)
	case "fetch":
		handleFetch(appInst, filtered, jsonOutput)
	case "list":
		handleList(appInst, jsonOutput)
	case "show":
		handleShow(appInst, filtered, jsonOutput)
	case "analyze":
		handleAnalyze(appInst, filtered, jsonOutput)
	case "report":
		handleReport(appInst, filtered, jsonOutput)
	case "import":
		handleImport(appInst, filtered)
	case "delete":
		handleDelete(appInst, filtered)
	case "server":
		handleServer(appInst, cfg, filtered)
	case "version":
		fmt.Println(cfg.App.Version)
	case "doctor":
		handleDoctor(store)
	default:
		printGlobalHelp()
		os.Exit(1)
	}
}

func handleInit(store storage.Store) {
	_, err := store.Mutate(func(db *storage.Database) error { return nil })
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error inicializando: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Base de datos inicializada.")
}

func handleFetch(appInst *app.App, args []string, jsonOut bool) {
	fs := NewFlagSet("fetch")
	platform := fs.String("platform", "", "Plataforma (instagram, x)")
	username := fs.String("username", "", "Nombre de usuario")
	_ = fs.Parse(args)
	if strings.TrimSpace(*platform) == "" || strings.TrimSpace(*username) == "" {
		fmt.Println("Uso: scout fetch --platform <plataforma> --username <usuario>")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := appInst.FetchAndAddProfile(ctx, *platform, *username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(p)
		return
	}
	printPersonFull(p)
}

func handleList(appInst *app.App, jsonOut bool) {
	list, err := appInst.ListPersons()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(list)
		return
	}
	if len(list) == 0 {
		fmt.Println("No hay perfiles.")
		return
	}
	fmt.Printf("%-14s %-20s %-10s %-12s %s\n", "ID", "Nombre", "Plataforma", "Usuario", "Seguidores")
	for _, p := range list {
		fmt.Printf("%-14s %-20s %-10s %-12s %d\n", p.ID, utils.Truncate(p.Name, 20), p.Platform, p.Username, p.Followers)
	}
}

func handleShow(appInst *app.App, args []string, jsonOut bool) {
	fs := NewFlagSet("show")
	key := fs.String("person", "", "ID, nombre o username")
	_ = fs.Parse(args)
	if strings.TrimSpace(*key) == "" {
		fmt.Println("Uso: scout show --person <id|nombre|username>")
		os.Exit(1)
	}
	p, err := appInst.GetPerson(*key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(p)
		return
	}
	printPersonFull(p)
}

func handleAnalyze(appInst *app.App, args []string, jsonOut bool) {
	fs := NewFlagSet("analyze")
	key := fs.String("person", "", "ID, nombre o username")
	_ = fs.Parse(args)
	if strings.TrimSpace(*key) == "" {
		fmt.Println("Uso: scout analyze --person <id|nombre|username>")
		os.Exit(1)
	}
	p, err := appInst.AnalyzePerson(*key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(p)
		return
	}
	printPersonFull(p)
}

func handleReport(appInst *app.App, args []string, jsonOut bool) {
	fs := NewFlagSet("report")
	key := fs.String("person", "", "ID, nombre o username")
	out := fs.String("out", "", "Archivo de salida (opcional)")
	_ = fs.Parse(args)
	if strings.TrimSpace(*key) == "" {
		fmt.Println("Uso: scout report --person <id> [--out archivo.md]")
		os.Exit(1)
	}
	report, err := appInst.ReportPerson(*key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if jsonOut {
		fmt.Println(report)
		return
	}
	if *out != "" {
		if err := os.WriteFile(*out, []byte(report), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error escribiendo archivo: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Reporte guardado en %s\n", *out)
	} else {
		fmt.Print(report)
	}
}

func handleImport(appInst *app.App, args []string) {
	fs := NewFlagSet("import")
	file := fs.String("file", "", "Archivo CSV")
	_ = fs.Parse(args)
	if strings.TrimSpace(*file) == "" {
		fmt.Println("Uso: scout import --file datos.csv")
		os.Exit(1)
	}
	f, err := os.Open(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	count, err := appInst.ImportCSV(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Importados %d posts.\n", count)
}

func handleDelete(appInst *app.App, args []string) {
	fs := NewFlagSet("delete")
	key := fs.String("person", "", "ID, nombre o username")
	_ = fs.Parse(args)
	if strings.TrimSpace(*key) == "" {
		fmt.Println("Uso: scout delete --person <id|nombre|username>")
		os.Exit(1)
	}
	deleted, err := appInst.DeletePerson(*key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Perfil eliminado: %s (%s)\n", deleted.Name, deleted.Username)
}

func handleServer(appInst *app.App, cfg config.Config, args []string) {
	fs := NewFlagSet("server")
	addr := fs.String("addr", cfg.Server.Addr, "Dirección")
	token := fs.String("token", cfg.Server.Token, "Token de autenticación")
	rateLimit := fs.Int("rate-limit", cfg.Server.RateLimit, "Rate limit por minuto")
	_ = fs.Parse(args)
	cfg.Server.Addr = *addr
	cfg.Server.Token = *token
	cfg.Server.RateLimit = *rateLimit
	if err := server.Run(cfg, appInst); err != nil {
		fmt.Fprintf(os.Stderr, "Error en servidor: %v\n", err)
		os.Exit(1)
	}
}

func handleDoctor(store storage.Store) {
	fmt.Println("🔍 Diagnosticando...")
	db, err := store.Load()
	if err != nil {
		fmt.Printf("Error cargando base de datos: %v\n", err)
	} else {
		fmt.Printf("Base de datos cargada: %d perfiles\n", len(db.Persons))
	}
	fmt.Println("Diagnóstico completado.")
}

func printPersonFull(p model.Person) {
	fmt.Println("============================================================")
	fmt.Printf("Perfil completo: %s (@%s) [%s]\n", p.Name, p.Username, p.Platform)
	if strings.TrimSpace(p.SourceURL) != "" {
		fmt.Printf("Origen: %s\n", p.SourceURL)
	}
	fmt.Printf("Bio: %s\n", bioOrFallback(p.Bio))
	fmt.Printf("Seguidores: %d | Siguiendo: %d\n", p.Followers, p.Following)
	fmt.Printf("Posts: %d | Raw posts: %d\n", len(p.Posts), p.RawPostsCount)
	fmt.Printf("Creado: %s\n", p.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("Actualizado: %s\n", p.UpdatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("Último fetch: %s\n", p.LastFetched.Format("2006-01-02 15:04"))
	fmt.Printf("Último análisis: %s\n", p.LastAnalyzed.Format("2006-01-02 15:04"))
	fmt.Println()

	fmt.Println("== Señales ==")
	fmt.Printf("Apertura: %.2f (conf %.2f)\n", p.Signals.Openness, p.Signals.OpennessConf)
	fmt.Printf("Responsabilidad: %.2f (conf %.2f)\n", p.Signals.Conscientiousness, p.Signals.ConscientiousnessConf)
	fmt.Printf("Extraversión: %.2f (conf %.2f)\n", p.Signals.Extraversion, p.Signals.ExtraversionConf)
	fmt.Printf("Amabilidad: %.2f (conf %.2f)\n", p.Signals.Agreeableness, p.Signals.AgreeablenessConf)
	fmt.Printf("Neuroticismo: %.2f (conf %.2f)\n", p.Signals.Neuroticism, p.Signals.NeuroticismConf)
	fmt.Printf("Sentiment: %.2f\n", p.Signals.Sentiment)
	fmt.Printf("Tokens: %d | Únicos: %d | Diversidad: %.2f\n", p.Signals.TokenCount, p.Signals.UniqueTokenCount, p.Signals.LexicalDiversity)
	fmt.Printf("Shannon: %.2f (norm %.2f) | Zipf slope: %.2f | Zipf fit: %.2f\n", p.Signals.ShannonEntropy, p.Signals.ShannonNormalized, p.Signals.ZipfSlope, p.Signals.ZipfFit)
	fmt.Printf("Bayes confidence: %.2f\n", p.Signals.BayesConfidence)
	fmt.Printf("Engagement medio: %.2f\n", p.Signals.Engagement)
	fmt.Printf("Engagement mediana: %.2f\n", p.Signals.EngagementMedian)
	fmt.Printf("Frecuencia de posts: %.2f posts/día\n", p.Signals.PostFrequency)
	fmt.Printf("Confianza intereses: %.2f\n", p.Signals.InterestConf)
	fmt.Printf("Confianza perfil: %.2f\n", p.Confidence)
	fmt.Printf("Perfil: %s - %s\n", p.Profile, p.Profile.Description())
	fmt.Printf("Comunicación sugerida: %s\n", p.Profile.CommunicationStyle())

	if len(p.Signals.Interests) > 0 {
		fmt.Println()
		fmt.Println("== Intereses detectados ==")
		for _, interest := range p.Signals.Interests {
			fmt.Printf("- %s\n", interest)
		}
	}
	if len(p.Signals.Evidence) > 0 {
		fmt.Println()
		fmt.Println("== Evidencia ==")
		for _, e := range p.Signals.Evidence {
			fmt.Printf("- %s\n", e)
		}
	}
	fmt.Println()
	fmt.Println("== Posts ==")
	if len(p.Posts) == 0 {
		fmt.Println("(sin posts)")
	} else {
		for i, post := range p.Posts {
			fmt.Printf("[%d] %s | likes:%d reposts:%d replies:%d\n", i+1, post.CreatedAt.Format("2006-01-02 15:04"), post.Likes, post.Reposts, post.Replies)
			if len(post.Hashtags) > 0 {
				fmt.Printf("    hashtags: %s\n", strings.Join(post.Hashtags, ", "))
			}
			if len(post.Mentions) > 0 {
				fmt.Printf("    mentions: %s\n", strings.Join(post.Mentions, ", "))
			}
			if len(post.Links) > 0 {
				fmt.Printf("    links: %s\n", strings.Join(post.Links, ", "))
			}
			if post.Language != "" {
				fmt.Printf("    language: %s\n", post.Language)
			}
			fmt.Printf("    %s\n", post.Text)
		}
	}
	fmt.Println("============================================================")
}

func printGlobalHelp() {
	fmt.Print(`LayerOne Scout - Modo de uso:
  scout <comando> [opciones]

Comandos:
  init                     Inicializa la base de datos
  fetch --platform <p> --username <u>  Obtiene perfil público
  list [--json]            Lista perfiles
  show --person <id> [--json]  Muestra detalles
  analyze --person <id> [--json]  Analiza/recalcula
  report --person <id> [--out <archivo>]  Genera informe
  import --file <csv>      Importa datos desde CSV
  delete --person <id>     Elimina un perfil
  server [--addr :8787] [--token x] [--rate-limit 120]  Inicia servidor HTTP
  version                  Muestra versión
  doctor                   Diagnóstico del sistema

Alias: ls = list, rm = delete

Opciones globales:
  --json                   Salida en JSON
  --debug                  Modo debug

Ejemplos:
  scout fetch --platform instagram --username usuario
  scout report --person usuario --out perfil.md
  scout server --addr :8080 --token secreto
`)
}
