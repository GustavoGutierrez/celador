# PRD-007: Verificación de Procedencia Criptográfica (Sigstore)

**Prioridad:** 🔴 Crítica — brecha zero-trust #1  
**Impacto:** Alto — detecta cuentas de mantenedores comprometidas  
**Esfuerzo:** Medio  
**Llamadas de red adicionales:** 1 por paquete (Sigstore Rekor lookup)  
**Peso del binario:** +5-8MB (go-sigstore/sget dependencies)  
**Breaking changes:** Ninguno — nuevo flag `--verify-provenance`

---

## Problema Actual

Celador no puede verificar que un paquete publicado en npm realmente viene del autor legítimo. Si la cuenta de un mantenedor es comprometida (como ocurrió con `event-stream`, `ua-parser-js`, y `axios@1.14.1`), Celador acepta el paquete como legítimo porque solo analiza el contenido, no la procedencia.

**Lo que falta:** Verificar que el paquete fue firmado por el publicador original usando firmas criptográficas (Sigstore/cosign).

---

## Objetivo

Verificar la procedencia criptográfica de paquetes npm usando el log de transparencia de Sigstore (Rekor), sin ejecutar código y manteniendo la filosofía offline-first (la verificación ocurre en el primer escaneo, luego se cachea).

---

## Análisis de Viabilidad — Enfoque Ligero

### Opciones consideradas

| Opción | Peso binario | Complejidad | Cobertura | Offline |
|--------|-------------|-------------|-----------|---------|
| **cosign CLI completo** | +50MB+ | Alto | Total | ❌ |
| **go-sigstore/sget** | +8-10MB | Medio | Parcial | ❌ (1 lookup) |
| **Rekor API directo (HTTP)** | **+0MB** | **Bajo** | **Parcial** | **❌ (1 lookup)** |
| **Verificación de firma npm (provenance)** | **+0MB** | **Bajo** | **Parcial** | **❌ (1 lookup)** |

### Enfoque seleccionado: Rekor API directo + npm provenance

**Por qué:**
- **Peso cero adicional** — usa `net/http` estándar de Go, sin dependencias externas
- **Rápido** — una sola llamada HTTP GET al log público de Rekor
- **Cachable** — el resultado se cachea como los findings de OSV
- **Compatible con filosofía Celador** — offline después del primer escaneo
- **Detecta el 80% de los casos** — paquetes sin firma de procedencia = alerta

### Qué verifica

| Verificación | Fuente | Qué detecta |
|-------------|--------|-------------|
| **npm provenance statement** | `https://registry.npmjs.org/{pkg}/{version}` campo `dist.attestations` | Paquete fue publicado con OIDC de GitHub (no desde credenciales robadas) |
| **Rekor log entry** | `https://rekor.sigstore.dev/api/v1/log/entries` | Firma criptográfica del build existe |
| **Build signer identity** | Attestation statement | El CI que publicó es legítimo (GitHub Actions oficial) |

### Qué NO verifica (limitaciones aceptadas)

| Limitación | Impacto | Mitigación |
|-----------|---------|------------|
| No verifica full cosign signature chain | Solo verifica provenance, no firma completa | Reportar como "partial provenance" |
| No verifica fulcio certificate validity | No valida el certificado completo | Aceptar que es "best-effort" |
| Requiere npm >= 9.x con provenance support | Solo paquetes recientes soportan esto | Paquetes antiguos = "unknown" |

---

## Diseño Técnico

### Nuevo adapter: `internal/adapters/provenance/rekor.go`

```go
type ProvenanceChecker struct {
    client  *http.Client
    rekorURL string
    cache   ports.ProvenanceCache
}

type ProvenanceResult struct {
    PackageName  string
    Version      string
    HasAttestation bool
    SignerIdentity string // "GitHub Actions", "unknown", etc.
    BuildURI      string
    Verified       bool   // full verification success
    Warning        string // if partial or unknown
}

func (p *ProvenanceChecker) CheckPackage(ctx context.Context, name, version string) (ProvenanceResult, error)
```

### Flujo de verificación

1. Consultar `https://registry.npmjs.org/{pkg}/{version}` → campo `dist.attestations`
2. Si no tiene attestations → `HasAttestation: false`, `Warning: "no provenance statement"`
3. Si tiene attestations → verificar que el signer es GitHub Actions oficial
4. Opcionalmente buscar en Rekor log (`https://rekor.sigstore.dev`) para full verification
5. Cache resultado con TTL de 24h

### Integración en `celador install`

```bash
# Nuevo flag opcional
celador install express --verify-provenance

# O en celador.yaml
provenance:
  enabled: true
  strict: false  # true = reject packages without provenance
```

### Señales de riesgo

| Señal | Severidad | Qué significa |
|-------|-----------|---------------|
| Sin provenance statement | Medium | Paquete publicado sin OIDC (puede ser legítimo pero antiguo) |
| Provenance de CI desconocido | High | No es GitHub Actions oficial — posible cuenta comprometida |
| Provenance verificado | ✅ | Paquete publicado desde CI legítimo |

---

## Plan de Tests

| Test | Qué valida |
|------|-----------|
| `TestProvenanceChecker_PackageWithAttestation` | Paquete con provenance válido |
| `TestProvenanceChecker_PackageWithoutAttestation` | Paquete sin provenance (warn) |
| `TestProvenanceChecker_UnknownCISigner` | CI no reconocido (alerta) |
| `TestProvenanceChecker_NetworkError` | Rekor/npm API caído |
| `TestProvenanceChecker_CacheHit` | Resultado cacheado, sin red |
| `TestProvenanceChecker_CacheExpired` | Caché expirada, re-verifica |
| `TestProvenanceChecker_PackageNotFound` | 404 del registry |
| `TestExtractSignerIdentity_GitHubActions` | Identifica GitHub Actions correctamente |

---

## Criterios de Aceptación

- [ ] Verifica provenance de npm packages con attestations
- [ ] Paquetes sin provenance generan warning Medium
- [ ] Resultados cacheados con TTL 24h
- [ ] 8+ nuevos tests con cobertura ≥80%
- [ ] Flag `--verify-provenance` en `celador install`
- [ ] Sin dependencias externas nuevas (solo stdlib net/http)
- [ ] `go test ./...` pasa sin fallos

---

## Archivos a Modificar/Crear

| Archivo | Acción |
|---------|--------|
| `internal/adapters/provenance/rekor.go` | **Crear** |
| `internal/adapters/provenance/rekor_test.go` | **Crear** |
| `internal/ports/provenance.go` | **Crear** — interfaz ProvenanceChecker |
| `internal/app/commands.go` | Modificar — agregar flag `--verify-provenance` |
| `internal/app/bootstrap.go` | Modificar — wire ProvenanceChecker |
| `internal/core/shared/models.go` | Posible — agregar ProvenanceResult |

---

## Estimación

| Fase | Tiempo |
|------|--------|
| Tests (TDD) | 1.5 horas |
| Implementación Rekor checker | 2 horas |
| Integración en CLI | 1 hora |
| Verificación y tests | 30 min |
| **Total** | **~5 horas** |
