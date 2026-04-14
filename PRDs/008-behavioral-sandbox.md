# PRP Ajustado — Sandbox de comportamiento portable, liviano y multiplataforma

## Resumen ejecutivo
La mejor alternativa **única** que maximiza portabilidad, bajo peso, rapidez y compatibilidad multiplataforma sin exigir configuración adicional del usuario es **un binario único en Go que embeba un motor JavaScript pequeño y portable como QuickJS, con instrumentación propia y política offline por defecto**.[cite:54][cite:60][cite:56]

Frente a `goja`, QuickJS ofrece soporte mucho más moderno del lenguaje, incluyendo módulos y gran parte de ES2023, además de ser muy pequeño y embebible, con una huella de código especialmente reducida y arranque muy rápido.[cite:54][cite:60] Frente a Docker, Bubblewrap, NsJail o gVisor, esta opción sacrifica parte del aislamiento a nivel de sistema, pero gana claramente en experiencia de instalación, peso, velocidad de arranque y funcionamiento uniforme en Linux, macOS y Windows.[cite:34][cite:37][cite:48]

La recomendación del presente ajuste es: **cambiar el objetivo principal del PRP desde “sandbox fuerte del sistema” a “runtime aislado embebido, portable y offline-first”**, dejando el aislamiento por contenedores o herramientas del sistema como backend opcional futuro, no como requisito de la primera versión.[cite:34][cite:48][cite:54]

## Contexto del PRP original
El PRP original identifica correctamente que el análisis estático no detecta protestware, malware condicional, ofuscación multicapa ni comportamientos activados en runtime, y propone `goja` o Node aislado como aproximaciones para cerrar esa brecha.[cite:1]

También acierta al exigir operación offline, modo opt-in y scoring por señales de comportamiento, pero la propuesta original prioriza opciones que son más fuertes en Linux que en otros sistemas o que dependen de runtimes externos, lo que afecta la promesa de portabilidad sin configuración adicional para el usuario final.[cite:1][cite:34]

## Evaluación de alternativas

| Alternativa | Portabilidad | Peso | Velocidad | Compatibilidad JS | Config extra del usuario | Veredicto |
|---|---|---:|---:|---|---|---|
| `goja` embebido en Go | Muy alta[cite:52][cite:56] | Baja[cite:1][cite:52] | Alta[cite:52] | Media, centrado en ES5.1[cite:52] | No[cite:56] | Bueno pero corto para ecosistema moderno |
| QuickJS embebido en Go/Rust | Muy alta[cite:54][cite:60] | Muy baja[cite:54][cite:60] | Muy alta[cite:54][cite:60] | Alta, con módulos y ES2023[cite:54][cite:60] | No[cite:54] | **Mejor opción única** |
| Node + Docker/OCI | Alta funcional, no nativa homogénea[cite:40][cite:32] | Media/alta[cite:32][cite:40] | Media[cite:32] | Muy alta | Sí, requiere Docker[cite:40] | Potente pero no ideal para cero-config |
| Bubblewrap / NsJail | Linux-only[cite:34][cite:37] | Baja | Alta | Depende de Node externo | Sí, depende del SO/herramienta[cite:34] | Excelente en Linux, no único multiplataforma |
| gVisor | Limitada y más compleja[cite:48] | Media/alta[cite:48] | Media | Muy alta | Sí, requiere runtime/config[cite:48] | No para primera versión |
| `deno_core` / V8 embebido | Alta teórica | Alta[cite:58] | Buena | Muy alta | No | Demasiado pesado/complejo |

## Recomendación única
La mejor opción única es un **binario autocontenido** escrito en **Go** o **Rust** que embeba **QuickJS** como motor de ejecución, aplique una API mínima controlada y capture señales de comportamiento sospechoso sin exponer al código analizado el sistema operativo real.[cite:54][cite:60]

Go tiene una ventaja clara en distribución porque permite cross-compilar binarios para múltiples sistemas operativos de forma sencilla, lo que reduce fricción operativa para publicar releases en Linux, macOS y Windows.[cite:56] Rust también es una buena opción para binarios portables y eficientes, pero el PRP original ya está orientado al ecosistema Go, por lo que continuar con Go reduce el costo de integración y mantenimiento.[cite:1][cite:56]

## Por qué QuickJS encaja mejor
QuickJS fue diseñado para ser pequeño y embebible, con muy bajo tiempo de arranque y soporte moderno del lenguaje, lo que lo vuelve especialmente atractivo para correr ejecuciones breves e instrumentadas en un CLI de seguridad.[cite:54][cite:60]

A diferencia de `goja`, que declara foco en ECMAScript 5.1, QuickJS soporta módulos, async generators, proxies, BigInt y una porción mucho más actual del estándar, lo que mejora de forma importante la compatibilidad con paquetes modernos y reduce la necesidad de fallback inmediato.[cite:52][cite:54][cite:60]

A diferencia de Node real dentro de un contenedor, QuickJS no exige instalación previa de Docker ni runtimes del sistema, y tampoco hereda toda la superficie operativa de un entorno Node completo.[cite:32][cite:40][cite:54]

## Qué se gana y qué se pierde
### Ventajas
- Un solo binario por plataforma, sin dependencia de Docker, Node, Bubblewrap o NsJail.[cite:54][cite:56]
- Experiencia casi cero-config para el usuario final.[cite:56]
- Arranque extremadamente rápido, útil para escanear muchos paquetes o hooks de instalación.[cite:54][cite:60]
- Mejor compatibilidad moderna que `goja`.[cite:52][cite:54]
- Operación offline natural, porque todo va embebido en el binario.[cite:54]

### Desventajas
- No reemplaza el aislamiento fuerte del sistema operativo para código verdaderamente hostil.[cite:34][cite:48]
- No ejecuta addons nativos de Node ni reproduce el runtime completo de Node.[cite:1]
- Algunas APIs específicas de Node deben ser simuladas o interceptadas manualmente.

## Ajuste estratégico recomendado
La arquitectura debe cambiar de “sandbox del sistema primero” a “**micro-runtime embebido primero**”. El objetivo no sería ejecutar un paquete Node completo como si estuviera en producción, sino ejecutar el código publicado bajo un runtime controlado que permita detectar patrones de comportamiento de riesgo de forma consistente en todos los sistemas operativos.[cite:1][cite:54]

Eso preserva mucho mejor los objetivos que priorizaste: **portabilidad**, **multiplataforma**, **peso bajo**, **rapidez** y **mínima configuración del usuario final**.[cite:54][cite:56]

## PRP ajustado

## PRD-008B: Behavioral Sandbox Portable con QuickJS embebido

### Prioridad
Alta — resuelve la brecha de análisis dinámico con la mejor relación entre portabilidad, peso y experiencia de uso.[cite:1][cite:54]

### Impacto
Alto — añade detección en runtime offline sin depender de herramientas del sistema ni runtimes externos.[cite:1][cite:54]

### Esfuerzo
Medio-Alto — más simple de distribuir que contenedores o backends OS-specific, pero requiere trabajo de instrumentación del runtime.[cite:54][cite:56]

### Problema
El análisis estático no detecta lógica maliciosa activada en runtime, pero las soluciones basadas en Node del sistema, contenedores o herramientas Linux-only comprometen la portabilidad o exigen configuración adicional al usuario final.[cite:1][cite:34][cite:40]

### Objetivo
Construir un runner de comportamiento offline, portable y multiplataforma que se distribuya como binario único y ejecute paquetes JavaScript en un entorno embebido e instrumentado para detectar señales de riesgo de seguridad.[cite:54][cite:56]

### Propuesta técnica
Implementar un `SandboxRunner` en Go que embeba QuickJS y cargue el código del paquete dentro de un runtime controlado con APIs mínimas y shims instrumentados.[cite:54][cite:60][cite:56]

### Principios de diseño
- Binario único por plataforma, sin dependencias externas en tiempo de ejecución.[cite:56]
- Offline por defecto, sin egress de red.
- Runtime embebido y controlado, no Node del sistema.
- Política de capacidades mínimas: solo se expone al script lo necesario para el análisis.
- Misma semántica funcional base en Linux, macOS y Windows.[cite:56]
- Opt-in en la primera versión, igual que el PRP original propone para el sandbox dinámico.[cite:1]

### Alcance funcional
- Cargar archivos JS del paquete y resolver imports ES modules compatibles con QuickJS.[cite:54][cite:60]
- Interceptar intentos de red, lectura de entorno, acceso a filesystem lógico, timers persistentes, evaluación dinámica y patrones de dropper.[cite:1]
- Detectar señales compuestas y producir un score de sospecha.[cite:1]
- Ejecutar todo con timeout estricto y límites internos de memoria/CPU cuando el embedding lo permita.[cite:54]

### No objetivos de la primera versión
- Compatibilidad total con Node.js.
- Soporte para addons nativos `.node`.[cite:1]
- Reproducción exacta del runtime de producción.
- Aislamiento tipo contenedor o syscall sandboxing a nivel kernel.[cite:34][cite:48]

### Arquitectura propuesta
```text
internal/adapters/sandbox/
├── runner.go              # interfaz principal
├── quickjs_engine.go      # binding y lifecycle del runtime
├── module_loader.go       # resolución de módulos del paquete
├── bootstrap.js           # bootstrap instrumentado
├── shims/
│   ├── http.js            # shim que reporta intentos de red
│   ├── fs.js              # shim de filesystem lógico
│   ├── process.js         # shim de env/platform
│   ├── timers.js          # monitoreo de loops/timers
│   └── eval.js            # hooks de eval/Function
├── monitor.go             # colector de eventos
├── signals.go             # scoring de riesgo
├── limits.go              # timeout y límites
└── fixtures/              # paquetes de prueba
```

### Interfaz principal
```go
type SandboxRunner interface {
    Run(ctx context.Context, pkgPath string, opts RunOptions) (*SandboxResult, error)
}

type RunOptions struct {
    Timeout time.Duration
    Strict bool
    Offline bool
    EntryStrategy string
}

type SandboxResult struct {
    Engine          string
    Executed        bool
    Duration        time.Duration
    TimedOut        bool
    ModuleFormat    string
    NetworkAttempts []string
    EnvReads        []string
    FileReads       []string
    FileWrites      []string
    DynamicExec     []string
    Timers          []string
    Warnings        []string
    SuspiciousScore int
    Verdict         string
}
```

### Estrategia de ejecución
1. Leer `package.json` para inferir entrypoint, formato y señales iniciales.[cite:1]
2. Resolver el archivo principal del paquete.
3. Crear runtime QuickJS nuevo por ejecución.[cite:54][cite:60]
4. Inyectar objetos globales y módulos shim controlados.
5. Ejecutar bootstrap con timeout.
6. Registrar eventos sospechosos.
7. Calcular score y emitir resultado.
8. Destruir runtime completamente.

### Política de APIs expuestas
El runtime no debe exponer acceso real a la red, sistema operativo ni filesystem completo. En su lugar, se expondrán stubs o shims que:

- registren intención de uso,
- devuelvan errores controlados,
- permitan inferir comportamiento sin conceder privilegios reales.

Ejemplos:
- `fetch()` o `http.request()` registran “network attempt” y fallan en modo offline.
- `process.env` devuelve un mapa vacío o un subconjunto sintético, registrando qué claves intenta leer el paquete.[cite:1]
- `fs.readFile()` solo puede leer el árbol lógico del paquete ya cargado para análisis.
- `eval` y `Function` generan señal explícita.[cite:1]

### Heurísticas iniciales
- lectura de variables de entorno,[cite:1]
- import o uso de primitivas de red,[cite:1]
- timers repetitivos o loops de larga duración,[cite:1]
- construcción dinámica de código,
- cadenas de decodificación en runtime,
- escritura + ejecución lógica posterior,
- fingerprinting del entorno.

### Compatibilidad multiplataforma
La compatibilidad se logra porque QuickJS es embebible y el binario principal puede compilarse para Linux, macOS y Windows sin exigir herramientas adicionales en la máquina del usuario.[cite:54][cite:56][cite:60]

La promesa debe ser “mismo CLI, mismo comportamiento base, mismo formato de salida” en los tres sistemas operativos, con diferencias mínimas solo en empaquetado de release.[cite:56]

### Distribución
- `celador-linux-amd64`
- `celador-linux-arm64`
- `celador-darwin-amd64`
- `celador-darwin-arm64`
- `celador-windows-amd64.exe`
- `celador-windows-arm64.exe` si aplica.[cite:56]

No debe requerirse Docker, Node ni configuración de sandbox del sistema para la primera versión.[cite:40][cite:56]

### Roadmap de backends futuros
Una vez estabilizado el runner portable, pueden añadirse backends opcionales:
- `node-oci` para máxima fidelidad,
- `linux-bwrap` para hardening adicional en Linux,
- `strict-host` para entornos enterprise.[cite:34][cite:37][cite:40]

Estos backends no sustituyen al runner portable; lo complementan.

### CLI propuesta
```bash
celador install express --sandbox-scan
celador scan ./node_modules/lodash --sandbox-engine quickjs
```

```yaml
sandbox:
  enabled: true
  engine: quickjs
  timeout: 5s
  offline: true
  strict: false
```

### Criterios de aceptación
- Funciona en Linux, macOS y Windows con binarios nativos por plataforma.[cite:56]
- No requiere instalar Docker, Node ni utilidades del sistema.[cite:40][cite:56]
- Mantiene operación offline total.[cite:1]
- Detecta las señales principales definidas por el PRP original.[cite:1]
- Añade mejor compatibilidad moderna que la alternativa `goja`.[cite:52][cite:54][cite:60]
- Mantiene tiempo de arranque muy bajo y costo operacional mínimo.[cite:54][cite:60]

### Riesgos y mitigaciones

| Riesgo | Impacto | Mitigación |
|---|---|---|
| Compatibilidad incompleta con APIs Node | Alto | Definir claramente alcance de primera versión y usar shims enfocados al análisis, no a ejecución completa |
| Paquetes que dependan de addons nativos | Alto | Reportar “unsupported native addon” y caer a análisis estático mejorado[cite:1] |
| Evasión por diferencias entre QuickJS y Node | Medio | Priorizar señales de intención y añadir backend opcional Node más adelante |
| Complejidad de bindings | Medio | Encapsular motor detrás de interfaz limpia y tests de compatibilidad |

### Tests propuestos
- `TestQuickJSRunner_BenignPackage`
- `TestQuickJSRunner_ESModulePackage`
- `TestQuickJSRunner_NetworkAttempt`
- `TestQuickJSRunner_EnvRead`
- `TestQuickJSRunner_DynamicEval`
- `TestQuickJSRunner_TimerLoop`
- `TestQuickJSRunner_ObfuscatedLoader`
- `TestQuickJSRunner_UnsupportedNativeAddon`
- `TestQuickJSRunner_WindowsParity`
- `TestQuickJSRunner_MacParity`
- `TestQuickJSRunner_LinuxParity`

### Veredicto del ajuste
Si la prioridad #1 es **una sola estrategia portable, liviana, rápida y realmente multiplataforma**, la mejor decisión no es Docker, ni Bubblewrap, ni NsJail, ni siquiera `goja` como primera elección. La mejor base es **QuickJS embebido dentro de un binario Go** con runtime controlado e instrumentado.[cite:52][cite:54][cite:56][cite:60]

Ese enfoque no ofrece el aislamiento kernel-level de un contenedor, pero sí entrega la mejor relación entre cobertura práctica, peso, velocidad, facilidad de distribución y cero fricción para el usuario final, que es exactamente la prioridad definida en este ajuste.[cite:54][cite:56]
