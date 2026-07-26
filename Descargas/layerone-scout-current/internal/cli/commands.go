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
		fmt.Fprintf(os.Stderr, "Error cargando config: %v
", err)
		os.Exit(1)
	}
	fetcher.Init()

	cmd := args[1]
	switch cmd {
	case "ls", "list":
		cmd = "list"
	case "rm", "delete":
		cmd = "delete"
	}

	store, err := storage.NewJSONStore(cfg.Storage.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error abriendo store: %v
", err)
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
		fmt.Fprintf(os.Stderr, "Error inicializando: %v
", err)
		os.Exit(1)
	}
	fmt.Println("Base de datos inicializada.")
}

func handleFetch(appInst *app.App, args []string, jsonOut bool) {
	fs := NewFlagSet("fetch")
	platform := fs.String("platform", "", "Plataforma (instagram, x)")
	username := fs.String("username", "", "Nombre de usuario")
	fs.Parse(args)
	if *platform == "" || *username == "" {
		fmt.Println("Uso: scout fetch --platform <plataforma> --username <usuario>")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := appInst.FetchAndAddProfile(ctx, *platform, *username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v
", err)
		os.Exit(1)
	}
	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(p)
	} else {
		fmt.Printf("Perfil de %s (%s) importado y analizado.
", p.Name, p.Username)
	}
}

func handleList(appInst *app.App, jsonOut bool) {
	list, err := appInst.ListPersons()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v
", err)
		os.Exit(1)
	}
	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(list)
		return
	}
	if len(list) == 0 {
		fmt.Println("No hay perfiles.")
		return
	}
	fmt.Printf("%-14s %-20s %-10s %-12s %s
", "ID", "Nombre", "Plataforma", "Usuario", "Seguidores")
	for _, p := range list {
		fmt.Printf("%-14s %-20s %-10s %-12s %d
", p.ID, utils.Truncate(p.Name, 20), p.Platform, p.Username, p.Followers)
	}
}

func handleShow(appInst *app.App, args []string, jsonOut bool) {
	fs := NewFlagSet("show")
	key := fs.String("person", "", "ID, nombre o username")
	fs.Parse(args)
	if *key == "" {
		fmt.Println("Uso: scout show --person <id|nombre|username>")
		os.Exit(1)
	}
	p, err := appInst.GetPerson(*key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v
", err)
		os.Exit(1)
	}
	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(p)
	} else {
		fmt.Printf("ID: %s
", p.ID)
		fmt.Printf("Nombre: %s
", p.Name)
		fmt.Printf("Usuario: %s (%s)
", p.Username, p.Platform)
		fmt.Printf("Bio: %s
", p.Bio)
		fmt.Printf("Seguidores: %d, Siguiendo: %d
", p.Followers, p.Following)
		fmt.Printf("Posts: %d
", len(p.Posts))
		fmt.Printf("Último análisis: %s
", p.LastAnalyzed.Format("2006-01-02 15:04"))
	}
}

func handleAnalyze(appInst *app.App, args []string, jsonOut bool) {
	fs := NewFlagSet("analyze")
	key := fs.String("person", "", "ID, nombre o username")
	fs.Parse(args)
	if *key == "" {
		fmt.Println("Uso: scout analyze --person <id|nombre|username>")
		os.Exit(1)
	}
	p, err := appInst.AnalyzePerson(*key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v
", err)
		os.Exit(1)
	}
	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(p)
	} else {
		fmt.Printf("Análisis completado para %s (%s)
", p.Name, p.Username)
	}
}

func handleReport(appInst *app.App, args []string, jsonOut bool) {
	fs := NewFlagSet("report")
	key := fs.String("person", "", "ID, nombre o username")
	out := fs.String("out", "", "Archivo de salida (opcional)")
	fs.Parse(args)
	if *key == "" {
		fmt.Println("Uso: scout report --person <id> [--out archivo.md]")
		os.Exit(1)
	}
	report, err := appInst.ReportPerson(*key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v
", err)
		os.Exit(1)
	}
	if jsonOut {
		fmt.Println(report)
		return
	}
	if *out != "" {
		if err := os.WriteFile(*out, []byte(report), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error escribiendo archivo: %v
", err)
			os.Exit(1)
		}
		fmt.Printf("Reporte guardado en %s
", *out)
	} else {
		fmt.Print(report)
	}
}

func handleImport(appInst *app.App, args []string) {
	fs := NewFlagSet("import")
	file := fs.String("file", "", "Archivo CSV")
	fs.Parse(args)
	if *file == "" {
		fmt.Println("Uso: scout import --file datos.csv")
		os.Exit(1)
	}
	f, err := os.Open(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v
", err)
		os.Exit(1)
	}
	defer f.Close()
	count, err := appInst.ImportCSV(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v
", err)
		os.Exit(1)
	}
	fmt.Printf("Importados %d posts.
", count)
}

func handleServer(appInst *app.App, cfg config.Config, args []string) {
	fs := NewFlagSet("server")
	addr := fs.String("addr", cfg.Server.Addr, "Dirección")
	token := fs.String("token", cfg.Server.Token, "Token de autenticación")
	rateLimit := fs.Int("rate-limit", cfg.Server.RateLimit, "Rate limit por minuto")
	fs.Parse(args)
	cfg.Server.Addr = *addr
	cfg.Server.Token = *token
	cfg.Server.RateLimit = *rateLimit
	if err := server.Run(cfg, appInst); err != nil {
		fmt.Fprintf(os.Stderr, "Error en servidor: %v
", err)
		os.Exit(1)
	}
}

func handleDoctor(store storage.Store) {
	fmt.Println("🔍 Diagnosticando...")
	db, err := store.Load()
	if err != nil {
		fmt.Printf(" Error cargando base de datos: %v
", err)
	} else {
		fmt.Printf("Base de datos cargada: %d perfiles
", len(db.Persons))
	}
	fmt.Println(" Diagnóstico completado.")
}

func printGlobalHelp() {
	fmt.Println(`LayerOne Scout - Mkt Uso:
  scout <comando> [opciones]

Comandos:
  init                     Inicializa la base de datos
  fetch --platform <p> --username <u>  Obtiene perfil público
  list [--json]            Lista perfiles
  show --person <id> [--json]  Muestra detalles
  analyze --person <id> [--json]  Analiza/recalcula
  report --person <id> [--out <archivo>]  Genera informe
  import --file <csv>      Importa datos desde CSV
  server [--addr :8787] [--token x] [--rate-limit 120]  Inicia servidor HTTP
  version                  Muestra versión
  doctor                   Diagnóstico del sistema

Alias: ls = list, rm = delete (próximamente)

Opciones globales:
  --json                   Salida en JSON
  --debug                  Modo debug

Ejemplos:
  scout fetch --platform instagram --username usuario
  scout report --person usuario --out perfil.md
  scout server --addr :8080 --token secreto
`)
}
