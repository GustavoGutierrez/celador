# PRD-004: Salida SARIF para Integración CI/CD

**Prioridad:** 🟡 Media  
**Impacto:** Medio — abre puerta a integración con GitHub Code Scanning, GitLab  
**Esfuerzo:** Bajo  
**Llamadas de red adicionales:** Cero  
**Peso del binario:** Sin cambio  
**Breaking changes:** Ninguno — nuevo flag `--format sarif`

---

## Problema Actual

Celador solo produce salida en texto y JSON genérico. No puede integrarse con:
- GitHub Code Scanning (requiere SARIF)
- GitLab Code Quality (requiere SARIF o formato específico)
- Azure DevOps Security Reviews (acepta SARIF)

Sin esto, Celador no puede ser un gate de seguridad en CI/CD.

---

## Objetivo

Agregar formato de salida SARIF v2.1.0 para que los hallazgos de `celador scan` puedan ser consumidos por plataformas de CI/CD.

---

## Diseño Técnico

### Nuevo flag en `celador scan`

```bash
celador scan --format sarif > results.sarif
```

### Implementación

**Nuevo archivo:** `internal/adapters/output/sarif.go`

```go
func ToSARIF(findings []shared.Finding, rules []shared.RuleConfig) *SARIFReport
```

**Estructura SARIF mínima:**
```json
{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [{
    "tool": {
      "driver": {
        "name": "Celador",
        "version": "0.3.2",
        "informationUri": "https://github.com/GustavoGutierrez/celador"
      }
    },
    "results": [
      {
        "ruleId": "GHSA-xxxx",
        "level": "error",
        "message": { "text": "Vulnerability in lodash" },
        "locations": [{ "physicalLocation": { "artifactLocation": { "uri": "package-lock.json" } } }]
      }
    ]
  }]
}
```

### Mapeo de severidad

| Celador | SARIF `level` |
|---------|---------------|
| `critical` | `error` |
| `high` | `error` |
| `medium` | `warning` |
| `low` | `note` |

---

## Plan de Tests

| Test | Qué valida |
|------|-----------|
| `TestToSARIF_EmptyFindings` | Sin hallazgos = results vacío |
| `TestToSARIF_SingleFinding` | Un hallazgo OSV se serializa correctamente |
| `TestToSARIF_MultipleFindings` | Múltiples hallazgos con diferentes severidades |
| `TestToSARIF_SeverityMapping` | Critical→error, High→error, Medium→warning, Low→note |
| `TestToSARIF_ValidJSON` | Salida es JSON válido conforme al schema SARIF |
| `TestToSARIF_RuleFindings` | Hallazgos de reglas (config) incluidos |

---

## Criterios de Aceptación

- [ ] Salida SARIF válida v2.1.0
- [ ] JSON válido conforme al schema oficial de SARIF
- [ ] GitHub Code Scanning puede consumir el archivo generado
- [ ] 6+ nuevos tests con cobertura ≥80%
- [ ] Flag `--format sarif` documentado en `celador scan --help`
- [ ] Sin cambios en comportamiento default (texto sigue siendo default)
- [ ] `go test ./...` pasa sin fallos

---

## Archivos a Modificar/Crear

| Archivo | Acción |
|---------|--------|
| `internal/adapters/output/sarif.go` | **Crear** |
| `internal/adapters/output/sarif_test.go` | **Crear** |
| `internal/app/commands.go` | Modificar — agregar flag `--format` a scan |
| `internal/core/shared/models.go` | Posible — agregar método ToSARIF si conviene |

---

## Estimación

| Fase | Tiempo |
|------|--------|
| Tests (TDD) | 1 hora |
| Implementación | 1-2 horas |
| Validación con schema SARIF | 30 min |
| **Total** | **~3 horas** |
