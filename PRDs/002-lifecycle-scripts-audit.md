# PRD-002: Auditoría Completa de Scripts de Lifecycle

**Prioridad:** 🟡 Media  
**Impacto:** Medio — cubre vector de ataque omitido actualmente  
**Esfuerzo:** Bajo  
**Llamadas de red adicionales:** Cero  
**Peso del binario:** Sin cambio  
**Breaking changes:** Ninguno

---

## Problema Actual

Celador solo detecta scripts `postinstall` en `package.json`. Los atacantes usan otros scripts de lifecycle que pasan desapercibidos:

| Script | Cuándo se ejecuta | Celador lo detecta |
|--------|-------------------|-------------------|
| `preinstall` | Antes de instalar dependencias | ❌ |
| `install` | Durante la instalación | ❌ |
| `prepare` | Antes de publicar y al instalar desde git | ❌ |
| `postinstall` | Después de instalar | ✅ |
| `prepublish` | Antes de publicar (npm legacy) | ❌ |

---

## Objetivo

Verificar **todos los scripts de lifecycle** de npm en `package.json`, no solo `postinstall`.

---

## Diseño Técnico

### Cambio en `internal/adapters/osv/registry_inspector.go`

**Actual:**
```go
if strings.Contains(text, "scripts") && strings.Contains(text, "postinstall") {
    assessment.Risk = maxSeverity(assessment.Risk, shared.SeverityMedium)
    assessment.ShouldPrompt = true
    assessment.Reasons = append(assessment.Reasons, "package defines install-time scripts")
}
```

**Después:**
```go
lifecycleScripts := []string{"preinstall", "install", "postinstall", "prepare"}
for _, script := range lifecycleScripts {
    if strings.Contains(text, script) {
        // ... misma lógica, reportando cuál script específico
    }
}
```

### Scripts a verificar

| Script | Severidad | Razón |
|--------|-----------|-------|
| `preinstall` | Medium | Se ejecuta antes de cualquier otra acción |
| `install` | Medium | Se ejecuta durante la instalación de dependencias |
| `postinstall` | Medium | Ya se detecta — se mantiene |
| `prepare` | Low | Se ejecuta al instalar desde URL de git (menos común) |
| `prepublish` | Low | Legacy de npm, raramente usado en paquetes modernos |

---

## Plan de Tests

| Test | Qué valida |
|------|-----------|
| `TestInspectPackage_PreinstallScript` | Detecta script `preinstall` |
| `TestInspectPackage_PrepareScript` | Detecta script `prepare` |
| `TestInspectPackage_MultipleLifecycleScripts` | Detecta múltiples scripts de lifecycle |
| `TestInspectPackage_NoLifecycleScripts` | Paquete sin scripts de lifecycle no alerta |

---

## Criterios de Aceptación

- [ ] Los 5 scripts de lifecycle son verificados
- [ ] Cada script se reporta individualmente en `assessment.Reasons`
- [ ] 4 nuevos tests con cobertura ≥80%
- [ ] Sin cambios en comportamiento para `postinstall` (retrocompatible)
- [ ] `go test ./...` pasa sin fallos

---

## Archivos a Modificar/Crear

| Archivo | Acción |
|---------|--------|
| `internal/adapters/osv/registry_inspector.go` | Modificar — ampliar check de scripts |
| `internal/adapters/osv/registry_inspector_test.go` | Ampliar — tests de nuevos scripts |

---

## Estimación

| Fase | Tiempo |
|------|--------|
| Tests (TDD) | 30 min |
| Implementación | 30 min |
| Verificación | 15 min |
| **Total** | **~1.5 horas** |
