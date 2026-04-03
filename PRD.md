# Documento de Requisitos del Producto (PRD) - v1.2

## 1. Visión General del Producto
**Nombre:** Celador CLI  
**Eslogan:** "La tranca de seguridad para tus dependencias." / "The security deadlock for your dependencies."  
**Propósito:** Herramienta ultrarrápida (estilo Vite) ejecutada desde la línea de comandos para auditar, bloquear y remediar proactivamente vulnerabilidades en dependencias, malas configuraciones de frameworks y fallos de seguridad en la cadena de suministro de npm, pnpm, Bun y Deno.

## 2. Arquitectura y Estándares de Ingeniería (Go)
Para garantizar la escalabilidad, la colaboración open-source y un mantenimiento impecable, el desarrollo del CLI se regirá por estándares estrictos:
- **Lenguaje:** Go (Golang) para compilar binarios estáticos ultrarrápidos y aprovechar la concurrencia nativa (`goroutines`).
- **Arquitectura Hexagonal (Ports & Adapters):** El código estará desacoplado. La lógica de negocio no dependerá del gestor de paquetes específico.
- **Librerías Recomendadas:**
  - CLI: `spf13/cobra` para enrutamiento de comandos y `spf13/viper` para configuración.
  - TUI: `charmbracelet/bubbletea` para la interfaz y `charmbracelet/lipgloss` para el estilo.
  - Testing: `stretchr/testify` para aserciones robustas.
- **Spec Driven Development (SDD):** El desarrollo estará estrictamente guiado por especificaciones. Se deben escribir primero las especificaciones (*Specs*) y los tests de comportamiento antes de la implementación funcional.
- **Prácticas de Código:** 
  - Todo el código fuente, comentarios y documentación interna estarán obligatoriamente en **Inglés**.
  - Desarrollo guiado por pruebas (Unit Testing robusto) cubriendo todos los dominios.
- **Motor de Interfaz (TUI):** Construido con `Charmbracelet Bubble Tea` para una Experiencia de Desarrollador (DX) hermosa e interactiva.
- **Fuente de Datos (Vulnerabilidades):** API oficial y gratuita de OSV.dev mediante peticiones por lotes (`querybatch`).
- **Caché Inteligente y Persistente:**
    - **Caché por Hash de Lockfile:** Genera un hash del archivo de bloqueo (`package-lock.json`, etc.). Si el hash no ha cambiado, devuelve el resultado cacheado en <50ms, evitando peticiones de red.
    - **Caché de API con TTL:** La caché de respuestas de la API OSV tendrá un TTL (Time-To-Live) configurable para modo offline y para reducir la latencia en ejecuciones repetidas.

### 2.1. Integración con OSV.dev (Ejemplo de Código)
Para consumir la API de OSV.dev eficientemente y sin costo, se usará el endpoint de `querybatch`. Aquí un ejemplo del mecanismo de integración en Go:

```go
package osv

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// OSVQuery representa la consulta para un paquete específico
type OSVQuery struct {
	Package OSVPackage `json:"package"`
	Version string     `json:"version"`
}

type OSVPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

// OSVBatchRequest es el payload para enviar múltiples consultas a la vez
type OSVBatchRequest struct {
	Queries []OSVQuery `json:"queries"`
}

// CheckVulnerabilities realiza una petición por lotes a la API gratuita de OSV
func CheckVulnerabilities(queries []OSVQuery) (*http.Response, error) {
	reqBody := OSVBatchRequest{Queries: queries}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// Usar el endpoint oficial de OSV.dev
	resp, err := http.Post("https://api.osv.dev/v1/querybatch", "application/json", bytes.NewBuffer(jsonData))
	return resp, err
}
```

## 3. Comandos Principales y Flujos de Usuario

### 3.1. `celador init` (Inicialización Guiada)
Comando único para integrar Celador en un proyecto existente.
- Detecta automáticamente el gestor de paquetes y los frameworks.
- Inyecta reglas de hardening en `.npmrc`, `bunfig.toml`, etc. (Sección 4.1).
- Crea el archivo `AGENTS.md` con directrices para IA (Sección 5.1).
- **Integración con Git Hooks:** Pregunta interactivamente si el usuario desea instalar `husky` y configurar un hook `pre-commit` para ejecutar `celador scan --staged`, asegurando que no se introduzcan vulnerabilidades en el repositorio.

### 3.2. `celador install` (Zero-Trust Wrapper)
Actúa como un proxy inteligente antes de pasarle el trabajo al gestor de paquetes subyacente. 
- **Pre-flight Checks:** Antes de instalar, descarga los tarballs temporalmente y busca anomalías:
  - Riesgo ALTO: Peticiones de red combinadas con extracción de datos del entorno (`process.env`).
  - Riesgo MEDIO: Cadenas de texto ofuscadas (Long hex-encoded strings).
- **Prompt Interactivo TUI:** Si se detectan anomalías, detiene la instalación y pregunta al usuario si desea continuar.

### 3.3. Auditoría y Remediación Proactiva (`celador scan` & `celador fix`)
- **`scan`:** Analiza rápidamente el lockfile usando la caché y la API de OSV.dev.
- **`fix`:** Aplica parches sin romper el proyecto usando *Safe SemVer Bumps* o inyectando `"overrides"`/`"resolutions"`. 
- **Visualización del "Blast Radius":** Antes de aplicar un parche, muestra qué otras dependencias se verán afectadas por el cambio para dar contexto y confianza al desarrollador.
- **Gestión de Excepciones:** Soporta un archivo `.celadorignore` para que los equipos puedan ignorar vulnerabilidades específicas con una razón y una fecha de expiración (ej. `celador ignore CVE-XXXX --reason="Not exploitable"`).
- Soporta flag `--pr` para abrir *Pull Requests* automáticamente y `--diff` para mostrar un Diff visual antes de confirmar.

## 4. Hardening y Defensa Proactiva de Configuración

### 4.1. Reglas de Configuración Globales
Inyecta reglas estrictas en `.npmrc`, `bunfig.toml` o `deno.json`:
- `ignore-scripts=true`
- `save-exact=true`
- `trust-policy=no-downgrade`

### 4.2. Prevención Zero-Day y Hardening Local
- `minimumReleaseAge: 1440`: Previene la instalación de paquetes con menos de 24 horas de publicación. 
- Excluye frameworks de extrema confianza: `minimumReleaseAgeExclude: ["webpack", "react", "typescript", "vite", "next", "nuxt"]`.
- Valida que `package.json` tenga un campo `"engines"` estricto.
- Verifica y repara `.gitignore` / `.npmignore` agregando `.env.local`, `*.map.js`, `*.js.map` y `coverage/`.
- En proyectos **AWS SAM**, valida el `template.yaml` para asegurar que el `Runtime` sea correcto y rechaza políticas IAM excesivas (`AdministratorAccess`).

### 4.3. Framework Fingerprinting & Misconfigurations
Al detectar frameworks, Celador audita sus archivos de configuración específicos.
- **Arquitectura de Reglas Extensible:** La lógica de detección no estará codificada en el binario. Celador leerá archivos de reglas (`.yaml`) desde un directorio. Esto permite a la comunidad y a las empresas añadir soporte para nuevos frameworks o reglas personalizadas sin necesidad de actualizar el CLI.
- **Reglas Soportadas (Ejemplos):**
    - **Next.js:** Alerta CRÍTICA si en `.env` hay variables `NEXT_PUBLIC_*` con palabras como `SECRET` o `TOKEN`. Exige `poweredByHeader: false` y alerta sobre `remotePatterns` con dominios comodín `*`.
    - **Nuxt.js:** Valida que no haya secretos en `runtimeConfig.public`.
    - **SvelteKit:** Asegura que la protección CSRF nativa no esté desactivada.
    - **Vite / React / Vue:** Bloquea el build de producción si `build.sourcemap: true`.
    - **Strapi:** Bloquea el arranque si se usan claves criptográficas por defecto.
    - **Angular:** Exige `sourceMap: false` y `budgets` estrictos en producción.
    - **Astro:** Valida restricciones CORS severas si `output: "server"`.

### 4.4. Tailwind CSS v4 (Pure CSS) y XSS
- **Bloat / DoS:** Advierte si se usa `@source` apuntando a directorios masivos.
- **Riesgo XSS (Arbitrary Values):** Escanea archivos `.tsx`, `.vue` o `.svelte` buscando interpolación dinámica de clases arbitrarias de Tailwind (ej. `className={"bg-[" + userInput + "]"}`).

## 5. Integración y Documentación para Agentes de IA

### 5.1. Reglas de Inyección en Proyecto (`AGENTS.md`)
Celador inyectará automáticamente en la raíz del proyecto (en `AGENTS.md` y `CLAUDE.md`) el siguiente bloque para gobernar a los LLMs:

```markdown
<!-- celador:start -->
## Celador Supply Chain Security 
This project has been hardened against supply chain attacks using [Celador](https://github.com/GustavoGutierrez/celador).

### Rules for AI assistants and contributors
- **Never use `^` or `~`** in dependency version specifiers. Always pin exact versions.
- **Always commit the lockfile**. Never delete it or add it to `.gitignore`.
- **Install scripts are disabled**. If a new dependency requires a build step, it must be explicitly approved.
- **New package versions must be at least 24 hours old** (minimum release age gating).
- **No dynamic Tailwind classes:** Never use string interpolation inside Tailwind arbitrary values to prevent XSS.
- **No raw SQL interpolation:** Always use parameterized queries or an ORM.
- Prefer well-maintained packages with verified publishers and provenance on npmjs.com.
- Run `pnpm install` / `npm ci` with the lockfile present — never bypass it.
- **Use deterministic installs**: prefer `npm ci` or `pnpm install --frozen-lockfile` in CI.
- **Do not store secrets in plain text** in `.env` files committed to version control.
<!-- celador:end -->
```

### 5.2. Autogeneración de Documentación para LLMs (`llm.txt`)
Al finalizar la implementación del CLI, se debe generar y mantener actualizado un archivo `llm.txt` en la raíz del repositorio de Celador. Este archivo está diseñado específicamente para ser consumido por herramientas de IA (Claude, Cursor, Copilot, etc.) y debe contener:
- Una explicación exhaustiva de cómo funciona el CLI completo.
- Todos los comandos disponibles (`scan`, `install`, `fix`, `report`) con sus respectivos *flags*.
- Las variaciones exactas de comportamiento y comandos dependiendo de si el usuario final está en un entorno Node.js (npm), pnpm, Bun o Deno.
- Ejemplos claros de cómo un LLM debe sugerir la remediación de vulnerabilidades usando Celador.
- Contexto sobre cómo Celador maneja internamente las reglas de Zero-Trust para que el LLM no intente sobreescribir configuraciones de seguridad ya mitigadas por la herramienta.

### 5.3. README.md
Solo si necesitas saber de que se trata este proyecto puedes leer el archivo README.md, pero el código fuente, los comentarios y la documentación interna del proyecto DEBEN estar en inglés para asegurar la máxima colaboración open-source y claridad técnica.

## 6. Distribución y Compatibilidad Multiplataforma
Celador será una herramienta universal y fácil de instalar en cualquier entorno de desarrollo.
- **Binarios Precompilados:** El pipeline de CI/CD usará **GoReleaser** para generar binarios estáticos para:
    - **Linux:** x86_64, ARM64 (.deb, .rpm, .tar.gz)
    - **macOS:** x86_64, ARM64 (Apple Silicon)
    - **Windows:** x86_64, ARM64 (.zip, .msi)
- **Métodos de Instalación:**
    - **Homebrew (macOS/Linux):** `brew install celador`
    - **npm/pnpm/Bun (Wrapper):** `npm install -g celador-cli`
    - **Script Universal (Linux/macOS):** `curl -fsSL https://codexlighthouse.com/celador/install.sh | sh`
    - **Imagen Docker:** Una imagen oficial estará disponible en Docker Hub (`ggutierrez/celador`) y en GitHub Container Registry (`ghcr.io/GustavoGutierrez/celador`) para una integración perfecta en pipelines de CI/CD.

## 7. Estructura del Proyecto y Convenciones de Nomenclatura

### 7.1. Estructura del Proyecto (Standard Go Layout)
El proyecto seguirá el estándar de la comunidad Go para asegurar mantenibilidad y consistencia:
```text
celador/
├── cmd/
│   └── celador/       # Punto de entrada de la aplicación (main.go)
├── internal/          # Código privado de la aplicación (Hexagonal Architecture)
│   ├── core/          # Lógica de dominio (Entities, Use Cases)
│   ├── ports/         # Interfaces que definen entradas y salidas
│   └── adapters/      # Implementaciones concretas (OSV API, npm/pnpm CLI, BubbleTea UI)
├── pkg/               # Librerías públicas que podrían ser usadas por otros proyectos (ej. parser de dependencias)
├── configs/           # Archivos de configuración por defecto y plantillas (.yaml)
├── scripts/           # Scripts de build, instalación (ej. install.sh) y despliegue
├── test/              # Tests end-to-end (E2E) y datos de prueba mockeados
├── go.mod
├── go.sum
└── README.md
```

### 7.2. Convenciones de Nomenclatura
- **Carpetas/Paquetes:** Siempre en minúsculas, sin guiones bajos, de una sola palabra preferiblemente (ej. `scanner`, `config`, `tui`).
- **Archivos Go:** Nombres descriptivos en minúsculas, separados por guiones bajos si es necesario (ej. `osv_client.go`, `npm_adapter.go`). Los tests siempre deben llevar el sufijo `_test.go`.
- **Variables y Funciones Locales:** `camelCase` (ej. `packageVersion`, `parseLockfile()`).
- **Structs, Interfaces y Funciones Exportadas:** `PascalCase` para hacerlos públicos fuera del paquete (ej. `DependencyScanner`, `type ScanResult struct`).
- **Constantes:** `PascalCase` para exportadas, `camelCase` para internas. No usar ALL_CAPS (ej. `const DefaultTimeout = 30`).
- **Archivos de Configuración:** `kebab-case` o `snake_case` (ej. `.celador-config.yaml`).

## 8. Directrices para Agentes IA y Skills (Buenas Prácticas en Go)
Todos los asistentes de IA o desarrolladores que contribuyan a este código base **DEBEN** utilizar los *Skills* configurados para este proyecto relacionados con buenas prácticas y arquitectura en Go. Específicamente:
- **`golang-patterns`**: Aplicar convenciones idiomáticas de Go, manejo de errores robusto, early returns, e interfaces implícitas.
- **`golang-pro`**: Utilizar para la implementación concurrente (Goroutines/Channels para escanear múltiples paquetes simultáneamente sin bloquear el hilo principal), uso correcto de Genéricos, optimización de memoria e integraciones de red limpias para llamadas a la API de OSV.
- Se debe asegurar que cualquier código generado pase por un pipeline estricto de `go fmt`, `go vet` y se analice usando herramientas como `golangci-lint` para garantizar que la calidad del código se mantiene en los estándares más altos.
