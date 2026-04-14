# PRD-008: Sandbox de Comportamiento Ligero (Node.js Isolado)

**Prioridad:** 🟡 Media — brecha zero-trust #2  
**Impacto:** Alto — detecta protestware, malware condicional, ofuscación multi-capa  
**Esfuerzo:** Alto  
**Llamadas de red adicionales:** 0 (totalmente offline)  
**Peso del binario:** +0MB (usa `node` del sistema)  
**Breaking changes:** Ninguno — nuevo flag `--sandbox-scan`

---

## Problema Actual

El análisis estático no puede detectar código que solo se vuelve malicioso bajo ciertas condiciones:

| Ataque | Por qué el análisis estático falla | Ejemplo |
|--------|-----------------------------------|---------|
| **Protestware** | El código es benigno hasta una condición específica | `colors.js` — bucle infinito solo si el hostname contiene "palantir" |
| **Malware condicional** | El payload se activa solo en producción | `ua-parser-js` — crypto-miner solo en ciertos entornos |
| **Ofuscación multi-capa** | `eval(atob(base64(x)))` evade patrones simples | Cadenas de transformación que solo se resuelven en runtime |
| **Código polimórfico** | Se transforma al ser leído | No hay patrón estático que coincida |

---

## Análisis de Opciones de Sandbox

### Opción 1: Node.js nativo en contenedor efímero

**Cómo funciona:**
1. Extraer el tarball del paquete
2. Crear un directorio temporal aislado
3. Ejecutar `node --eval "require('./index.js')"` con timeout de 5 segundos
4. Monitorear: procesos hijos, llamadas de red, acceso a filesystem, variables de entorno
5. Matar el proceso y reportar hallazgos

**Dependencias:** `node` del sistema (ya requerido por Celador para JS/TS workspaces)
**Peso adicional:** 0MB
**Complejidad:** Media

**Pros:**
- Detecta comportamiento real (no patrones estimados)
- Zero dependencias adicionales
- Portable (donde haya node, funciona)
- Offline completo

**Contras:**
- Riesgo de seguridad: ejecutar código no confiable
- Necesita aislamiento cuidadoso (sin acceso a red real, FS limitado)
- Puede ser lento (5-10s por paquete)
- No funciona sin node instalado

---

### Opción 2: Análisis estático avanzado (AST parsing)

**Cómo funciona:**
1. Parsear cada archivo JS/TS con un parser de JavaScript embebido
2. Construir el AST (Abstract Syntax Tree)
3. Analizar flujo de datos: ¿de dónde viene el argumento de eval()?
4. Detectar ofuscación encadenada: eval(atob(base64(x)))
5. Simular ejecución parcial sin ejecutar código

**Dependencias:** Parser JS embebido (~500KB minificado)
**Peso adicional:** +500KB
**Complejidad:** Muy alta

**Pros:**
- Seguro (no ejecuta código)
- Detecta ofuscación multi-capa
- Rápido una vez implementado

**Contras:**
- Parser JS completo en Go es muy complejo
- No detecta comportamiento condicional
- +500KB al binario
- Implementación de meses

---

### Opción 3: WebAssembly sandbox (goja)

**Cómo funciona:**
1. Usar `goja` (intérprete JavaScript en Go puro)
2. Ejecutar el código del paquete en un VM aislado
3. Interceptar todas las llamadas a `require()`, `process`, `fs`, `http`
4. Reportar comportamiento sospechoso

**Dependencias:** `github.com/dop251/goja` (~2MB)
**Peso adicional:** +2MB
**Complejidad:** Media-Alta

**Pros:**
- Seguro (VM aislada, sin acceso al sistema real)
- Go puro — sin dependencias externas al sistema
- Portable — no necesita node instalado
- Detecta comportamiento real

**Contras:**
- +2MB al binario
- No soporta módulos nativos (.node)
- No ejecuta C++ addons
- Puede tener incompatibilidades con JS moderno

---

### Opción 4: Node.js con `--inspect` y monitoreo de syscalls

**Cómo funciona:**
1. Ejecutar `node --inspect` en directorio temporal
2. Usar `strace` (Linux) o `dtrace` (macOS) para monitorear syscalls
3. Detectar: open(), connect(), execve(), kill()
4. Timeout de 5 segundos

**Dependencias:** `node` + `strace`/`dtrace` del sistema
**Peso adicional:** 0MB
**Complejidad:** Media

**Pros:**
- Comportamiento real a nivel de sistema
- Zero dependencias adicionales
- Detecta lo que el sandbox de VM puede no ver

**Contras:**
- `strace` no está disponible en todos los sistemas
- No portable a Windows sin WSL
- Más lento que goja

---

## Recomendación: Opción 1 + Opción 3 (híbrido)

**Estrategia:**
1. **Primero:** Intentar con `goja` (VM aislada, sin riesgos, +2MB)
2. **Si goja falla** (incompatibilidad con el JS): fallback a node sandbox con timeout
3. **Si no hay node disponible:** reportar "sandbox unavailable" y volver a análisis estático

**Justificación:**
- `goja` cubre el 70% de los paquetes (JS puro, sin C++ addons)
- Node sandbox cubre el 30% restante (nativos, addons, ESM moderno)
- Peso total: +2MB (goja) + 0MB (node del sistema)
- Portable: funciona sin node si el paquete es JS puro
- Seguro: goja es VM aislada, node sandbox tiene timeout + directorio temporal

---

## Diseño Técnico — goja sandbox

### Dependencia única

```bash
go get github.com/dop251/goja@latest
```

**Tamaño:** ~2MB en el binario  
**Mantenimiento:** Activo, 500+ stars, usado en producción por empresas

### Arquitectura

```
internal/adapters/sandbox/
├── goja_runner.go     — VM aislada con interceptores
├── node_runner.go     — Fallback a node con timeout
├── monitor.go         — Monitoreo de comportamiento
└── signals.go         — Definición de señales de riesgo
```

### goja_runner.go

```go
type SandboxResult struct {
    Executed        bool              // ¿Se ejecutó algo?
    Duration        time.Duration     // Tiempo de ejecución
    NetworkCalls    []NetworkAttempt  // Intentos de red
    FileAccess      []FileAttempt     // Accesos a FS
    EnvAccess       []string          // Variables de entorno leídas
    ChildProcesses  []string          // Procesos hijos intentados
    SuspiciousScore int               // Score de 0-100
    Warnings        []string          // Alertas generadas
}

func RunInGoja(ctx context.Context, packagePath string) (*SandboxResult, error)
```

### Interceptores en goja

| Función interceptada | Qué reporta | Ejemplo de alerta |
|---------------------|-------------|-------------------|
| `require('http')` | Intento de red | "Package attempted HTTP connection" |
| `require('https')` | Intento de red | "Package attempted HTTPS connection" |
| `require('fs')` | Acceso a filesystem | "Package read filesystem" |
| `require('child_process')` | Ejecución de procesos | "Package spawned child process" |
| `require('os')` | Info del sistema | "Package accessed OS info" |
| `process.env` | Variables de entorno | "Package read env vars" |
| `eval()` | Ejecución dinámica | "Package used eval() at runtime" |
| `new Function()` | Code injection | "Package created dynamic function" |
| `setInterval` con loop | Posible protestware | "Package created infinite timer" |

### Señales de riesgo

| Señal | Score | Ejemplo real |
|-------|-------|--------------|
| Intento de red a dominio externo | +30 | `ua-parser-js` crypto-miner |
| `process.env` leído + red | +40 | Exfiltración de secrets |
| `child_process.exec` | +50 | Ejecución de comandos |
| `eval()` con string dinámico | +20 | Ofuscación |
| `setInterval` infinito | +30 | `colors.js` protestware |
| Acceso a `~/.ssh/` | +50 | Robo de claves SSH |
| Write a temp + exec | +40 | Dropper pattern |
| Solo console.log | +0 | Paquete benigno |

### node_runner.go (fallback)

```go
func RunInNode(ctx context.Context, packagePath string) (*SandboxResult, error) {
    // Create temp dir with limited permissions
    // Run: node --eval "require('./index.js')" with 5s timeout
    // Monitor: /proc/PID/net/tcp (Linux) or netstat
    // Kill process after timeout
    // Parse results
}
```

---

## Integración en Celador

### Nuevos flags

```bash
# Escaneo con sandbox (instala + ejecuta en aislamiento)
celador install express --sandbox-scan

# Config en .celador.yaml
sandbox:
  enabled: true
  engine: goja  # goja | node | auto
  timeout: 5s
  strict: false  # true = reject packages with suspicious behavior
```

### Dónde se ejecuta

- En `celador install`: antes de delegar al package manager
- En `celador scan`: para paquetes ya instalados (analiza node_modules)
- NO se ejecuta por defecto (es opt-in por el costo de tiempo)

---

## Plan de Tests

| Test | Qué valida |
|------|-----------|
| `TestGojaRunner_BenignPackage` | Paquete benigno ejecuta sin alertas |
| `TestGojaRunner_NetworkAttempt` | Detecta intento de llamada HTTP |
| `TestGojaRunner_EnvAccess` | Detecta lectura de process.env |
| `TestGojaRunner_ChildProcess` | Detecta spawn de child_process |
| `TestGojaRunner_EvalAtRuntime` | Detecta eval() en ejecución |
| `TestGojaRunner_InfiniteLoop` | Timeout funciona, mata proceso |
| `TestGojaRunner_FileAccess` | Detecta lectura de filesystem |
| `TestGojaRunner_CompositeSignals` | Score compuesto por múltiples señales |
| `TestNodeRunner_Fallback` | Fallback a node cuando goja falla |
| `TestNodeRunner_Timeout` | Node sandbox respeta timeout |
| `TestSandboxIntegration_Protestware` | Detecta colors.js-like behavior |
| `TestSandboxIntegration_CleanPackage` | No falsos positivos en express/lodash |

---

## Criterios de Aceptación

- [ ] goja VM ejecuta paquetes JS puros en aislamiento
- [ ] Detecta al menos 8 tipos de comportamiento sospechoso
- [ ] Timeout de 5s respetado siempre
- [ ] Fallback a node cuando goja falla
- [ ] 12+ tests con cobertura ≥80%
- [ ] Flag `--sandbox-scan` documentado
- [ ] No ejecuta sandbox por defecto (opt-in)
- [ ] `go test ./...` pasa sin fallos
- [ ] Sin falsos positivos en express, lodash, react

---

## Archivos a Modificar/Crear

| Archivo | Acción |
|---------|--------|
| `internal/adapters/sandbox/goja_runner.go` | **Crear** |
| `internal/adapters/sandbox/node_runner.go` | **Crear** |
| `internal/adapters/sandbox/monitor.go` | **Crear** |
| `internal/adapters/sandbox/signals.go` | **Crear** |
| `internal/adapters/sandbox/goja_runner_test.go` | **Crear** |
| `internal/adapters/sandbox/node_runner_test.go` | **Crear** |
| `internal/adapters/sandbox/monitor_test.go` | **Crear** |
| `internal/ports/sandbox.go` | **Crear** — interfaz SandboxRunner |
| `internal/app/commands.go` | Modificar — agregar flag `--sandbox-scan` |
| `internal/app/bootstrap.go` | Modificar — wire SandboxRunner |

---

## Riesgos y Mitigación

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|-------------|---------|------------|
| goja no soporta ESM modules | Media | Alto | Fallback a node sandbox |
| goja no soporta syntax moderno (top-level await) | Media | Medio | Fallback a node sandbox |
| Ejecutar malware en sandbox escapa | Baja | Crítico | goja es VM pura — no puede escapar |
| Node sandbox no tiene strace en macOS | Alta | Medio | Usar node --inspect en lugar de strace |
| Falsos positivos en paquetes legítimos | Media | Alto | Whitelist de paquetes conocidos, thresholds ajustables |
| Performance degradation | Alta | Medio | Opt-in solamente, no default |

---

## Estimación

| Fase | Tiempo |
|------|--------|
| Tests (TDD) | 3 horas |
| goja sandbox core | 4 horas |
| node sandbox fallback | 3 horas |
| Signal system | 2 horas |
| Integración en CLI | 1 hora |
| Testing con paquetes reales | 2 horas |
| **Total** | **~15 horas** |

---

## Comparación de Complejidad con PRDs Anteriores

| PRD | Horas estimadas | Dependencias nuevas | Peso binario | Complejidad |
|-----|----------------|--------------------|--------------|-------------|
| PRD-001 (tarball) | 5h | 0 | 0 | ⭐⭐ |
| PRD-002 (lifecycle) | 1.5h | 0 | 0 | ⭐ |
| PRD-003 (typosquatting) | 4h | 0 | +2KB | ⭐⭐ |
| PRD-004 (SARIF) | 3h | 0 | 0 | ⭐ |
| PRD-005 (SBOM) | 3.5h | 0 | 0 | ⭐⭐ |
| PRD-006 (parallel) | 2.5h | 1 (errgroup) | 0 | ⭐ |
| **PRD-007 (provenance)** | **5h** | **0** | **0** | **⭐⭐** |
| **PRD-008 (sandbox)** | **15h** | **1 (goja)** | **+2MB** | **⭐⭐⭐⭐** |

---

## Veredicto

PRD-008 es **el PRD más complejo del roadmap** — 3x más esfuerzo que el siguiente más complejo.

**Vale la pena si:**
- Se quiere llegar a ~90%+ de detección de ataques (de ~70% actual)
- Se acepta +2MB al binario y un opt-in para el usuario
- Se quiere cerrar la brecha más grande contra Socket.dev

**No vale la pena si:**
- El binario debe mantenerse mínimo absoluto
- No se quiere riesgo de ejecutar código (aunque goja es VM aislada)
- El uso principal es offline total (goja funciona offline, node sandbox no)

**Recomendación:** Implementar como **feature experimental opt-in** primero, medir adopción, y decidir si promover a estable.
