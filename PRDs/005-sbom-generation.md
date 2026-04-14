# PRD-005: Generación de SBOM (SPDX)

**Prioridad:** 🟡 Media  
**Impacto:** Medio — requisito de cumplimiento 2024+  
**Esfuerzo:** Medio  
**Llamadas de red adicionales:** Cero  
**Peso del binario:** Sin cambio  
**Breaking changes:** Ninguno — nuevo flag `--sbom spdx`

---

## Problema Actual

Celador no genera Software Bill of Materials (SBOM). Sin SBOM no puede:
- Cumplir con requisitos de seguridad de gobierno (EO 14028 en EE.UU.)
- Integrarse con herramientas de gestión de vulnerabilidades enterprise
- Proveer inventario completo de dependencias para auditorías

---

## Objetivo

Generar SBOM en formato SPDX 2.3 a partir del lockfile del proyecto, sin llamadas de red adicionales.

---

## Diseño Técnico

### Nuevo flag en `celador scan`

```bash
celador scan --sbom spdx > sbom.spdx
celador scan --sbom spdx-json > sbom-spdx.json
```

### Implementación

**Nuevo archivo:** `internal/adapters/output/spdx.go`

```go
func ToSPDX(deps []shared.Dependency, ws shared.Workspace) *SPDXDocument
```

**Estructura SPDX mínima (tag-value):**
```
SPDXVersion: SPDX-2.3
DataLicense: CC0-1.0
SPDXID: SPDXRef-DOCUMENT
DocumentName: celador-scan
DocumentNamespace: https://celador.dev/spdx/abc123

## Creation Info
Creator: Tool: Celador-0.3.2
Created: 2026-04-13T22:00:00Z

## Packages
PackageName: lodash
SPDXID: SPDXRef-Package-lodash
PackageVersion: 4.17.21
PackageSupplier: NOASSERTION
PackageDownloadLocation: https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz

PackageName: express
SPDXID: SPDXRef-Package-express
PackageVersion: 4.18.2
...
```

### Datos disponibles (todo del lockfile, sin red)

| Campo SPDX | Fuente |
|-----------|--------|
| PackageName | `dep.Name` del lockfile |
| PackageVersion | `dep.Version` del lockfile |
| PackageDownloadLocation | Construido desde npm registry URL + nombre + versión |
| ExternalRef | `purl:pkg:npm/{name}@{version}` |

### Contenido del SBOM

| Sección | Contenido |
|---------|-----------|
| Document | Metadata del escaneo (tool, fecha, root) |
| Packages | Todas las dependencias del lockfile |
| Relationships | `DEPENDS_ON` entre paquete raíz y dependencias |

---

## Plan de Tests

| Test | Qué valida |
|------|-----------|
| `TestToSPDX_EmptyDeps` | Sin dependencias = SBOM con 0 paquetes |
| `TestToSPDX_SingleDep` | Un paquete genera package block correcto |
| `TestToSPDX_MultipleDeps` | Múltiples paquetes con purls correctos |
| `TestToSPDX_ValidTagValue` | Formato tag-value es parseable |
| `TestToSPDX_PackagePurl` | Purl generado correctamente (pkg:npm/lodash@4.17.21) |
| `TestToSPDX_DocumentMetadata` | Tool name, version, date incluidos |

---

## Criterios de Aceptación

- [ ] SBOM SPDX 2.3 válido y parseable
- [ ] Todos los campos obligatorios presentes
- [ ] Purls generados correctamente para cada dependencia
- [ ] 6+ nuevos tests con cobertura ≥80%
- [ ] Flag `--sbom spdx` documentado en `celador scan --help`
- [ ] Sin llamadas de red
- [ ] `go test ./...` pasa sin fallos

---

## Archivos a Modificar/Crear

| Archivo | Acción |
|---------|--------|
| `internal/adapters/output/spdx.go` | **Crear** |
| `internal/adapters/output/spdx_test.go` | **Crear** |
| `internal/app/commands.go` | Modificar — agregar flag `--sbom` a scan |

---

## Estimación

| Fase | Tiempo |
|------|--------|
| Tests (TDD) | 1 hora |
| Implementación | 2 horas |
| Validación formato SPDX | 30 min |
| **Total** | **~3.5 horas** |
