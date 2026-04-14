# PRD-006: Paralelizar Hidratación de Advisories OSV

**Prioridad:** 🟢 Rendimiento  
**Impacto:** Alto — reduce escaneos de 5-30s a 1-6s en proyectos vulnerables  
**Esfuerzo:** Bajo  
**Llamadas de red adicionales:** Cero (misma cantidad, pero en paralelo)  
**Peso del binario:** Sin cambio  
**Breaking changes:** Ninguno

---

## Problema Actual

Cuando OSV devuelve vulnerabilidades sin datos completos de `affected`, Celador consulta cada advisory individualmente en un **bucle secuencial**.

**Impacto:**
- 20 advisories × 500ms = **10 segundos** de espera
- 50 advisories × 500ms = **25 segundos** de espera
- El usuario percibe que Celador "está lento" o "colgado"

---

## Objetivo

Paralelizar las llamadas de hidratación de advisories usando `errgroup.Group` con límite de concurrencia, reduciendo el tiempo total de 5-10x.

---

## Diseño Técnico

### Cambio en `internal/adapters/osv/client.go`

**Actual (secuencial):**
```go
for _, vuln := range result.Vulns {
    if advisoryNeedsHydration(advisory) {
        if details, err := c.fetchAdvisory(ctx, advisory.ID); err == nil {
            advisory = details
        }
    }
}
```

**Después (paralelo con semáforo):**
```go
var mu sync.Mutex
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(10) // Máximo 10 requests concurrentes

for _, vuln := range result.Vulns {
    vuln := vuln // capture loop variable
    if advisoryNeedsHydration(vuln) {
        g.Go(func() error {
            details, err := c.fetchAdvisory(ctx, vuln.ID)
            if err == nil {
                mu.Lock()
                hydrated[vuln.ID] = details
                mu.Unlock()
            }
            return nil // No fallar el grupo por un advisory individual
        })
    }
}
_ = g.Wait() // Ignorar errores individuales (best-effort)
```

### Por qué `errgroup` y no goroutines sueltas

| Aspecto | Goroutines sueltas | `errgroup` |
|---------|-------------------|------------|
| Cancelación de contexto | Manual | Automática |
| Límite de concurrencia | Manual (buffered channel) | `SetLimit()` built-in |
| Wait | Manual (WaitGroup) | `Wait()` built-in |
| Error propagation | Manual | Automática |

### Límite de concurrencia

**10 concurrentes** como default:
- Suficiente para reducir 10s a ~1s
- No satura el cliente HTTP
- Respeta límites de rate de OSV API (~100 req/min free tier)

### Hacerlo configurable

```bash
# Environment variable para ajustar
CELADOR_OSV_HYDRATION_CONCURRENCY=20 celador scan
```

---

## Plan de Tests

| Test | Qué valida |
|------|-----------|
| `TestHydrateAdvisories_Parallel_FasterThanSequential` | 10 advisories se hidratan en <2s vs 10s secuencial |
| `TestHydrateAdvisories_RespectsConcurrencyLimit` | No más de N requests concurrentes |
| `TestHydrateAdvisories_SingleAdvisory` | Un advisory funciona sin panic |
| `TestHydrateAdvisories_NoneNeedHydration` | 0 advisories a hidratar = sin goroutines |
| `TestHydrateAdvisories_SomeFail_OthersSucceed` | Errores individuales no rompen el resto |
| `TestHydrateAdvisories_ContextCancellation` | Context cancelado detiene todas las goroutines |

---

## Criterios de Aceptación

- [ ] Hidratación de 10 advisories en <2s (vs 10s secuencial)
- [ ] Máximo N requests concurrentes (configurable, default 10)
- [ ] Fallos individuales no cancelan la hidratación del resto
- [ ] Context cancelado detiene todas las goroutines pendientes
- [ ] 6+ nuevos tests con cobertura ≥80%
- [ ] `go test ./...` pasa sin fallos
- [ ] `go test -race ./...` sin data races

---

## Archivos a Modificar/Crear

| Archivo | Acción |
|---------|--------|
| `internal/adapters/osv/client.go` | Modificar — paralelizar hidratación |
| `internal/adapters/osv/client_test.go` | Ampliar — tests de concurrencia |

---

## Dependencias

| Dependency | Versión | Razón |
|------------|---------|-------|
| `golang.org/x/sync` | `errgroup` | Ya es dependencia transitiva de otros paquetes |

Verificar con `go mod graph | grep errgroup`. Si no está presente, agregar:
```bash
go get golang.org/x/sync/errgroup
```

---

## Riesgos y Mitigación

| Riesgo | Probabilidad | Mitigación |
|--------|-------------|------------|
| Data race en mapa `hydrated` | Media | Mutex para writes concurrentes |
| Rate limiting de OSV API | Baja | Límite de 10 concurrentes está por debajo del límite free tier |
| Memory pressure con muchos advisories | Baja | errgroup.Wait() completa antes de continuar |

---

## Estimación

| Fase | Tiempo |
|------|--------|
| Tests (TDD) | 1 hora |
| Implementación | 1 hora |
| Test con `-race` | 30 min |
| **Total** | **~2.5 horas** |
