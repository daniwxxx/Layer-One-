# LayerOne Scout - Mkt Uso
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
