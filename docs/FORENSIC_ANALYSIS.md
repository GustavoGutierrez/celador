# Celador — Análisis Forense del Proyecto (Post-PRDs Completo)

**Fecha:** 14 de abril de 2026  
**Versión analizada:** v0.4.5  
**Metodología:** Revisión completa de código, análisis competitivo, perfilado de rendimiento

---

## Resumen Ejecutivo

Celador v0.4.5 es un **escáner de seguridad de cadena de suministro para JS/TS/Deno** que ha transitado de detección superficial (~10% de ataques conocidos) a **detección estática genuina (~70%+)** mediante la inspección de todos los archivos fuente del tarball, typosquatting offline, generación de SBOM/SARIF, y paralelización de rendimiento.

**Lo que es ahora:** Una herramienta de primera línea de defensa para desarrolladores y gate de seguridad en CI/CD.

**Lo que aún no es:** Una herramienta zero-trust plena (falta procedencia criptográfica y sandbox).

---

## 1. Estado de Cobertura de Detección

### 1.1 Ataques que Celador Detecta en v0.4.5

| Tipo de Ataque | Ejemplo Real | Detecta Celador | Cómo lo detecta |
|----------------|-------------|-----------------|-----------------|
| **CVEs conocidas** | Cualquier CVE en OSV | ✅ | API OSV batch |
| **eval() en archivos .js/.ts** | Crypto-miners inyectados | ✅ **(PRD-001)** | Patrón estático en archivos fuente |
| **new Function() en fuente** | Code injection dinámico | ✅ **(PRD-001)** | Patrón estático |
| **child_process.exec/spawn** | Ejecución de comandos del sistema | ✅ **(PRD-001)** | Patrón estático |
| **process.env + red en fuente** | Exfiltración de secrets | ✅ **(PRD-001)** | Co-ocurrencia de tokens |
| **https.request/fetch en fuente** | Comunicación C2 | ✅ **(PRD-001)** | Patrón estático |
| **Strings hex-encoded >80 chars** | Payloads ofuscados | ✅ **(PRD-001)** | Detección de secuencias hex contiguas |
| **Archivos .node no documentados** | Binarios nativos maliciosos | ✅ **(PRD-001)** | Detección de extensión |
| **Scripts de lifecycle (5 tipos)** | preinstall, install, postinstall, prepare, prepublish | ✅ **(PRD-002)** | Cada script reportado individualmente |
| **Typosquatting (distancia 1-2)** | lodahs, reacts, crossenv, axois | ✅ **(PRD-003)** | Levenshtein contra 100+ paquetes conocidos |
| **Configuración insegura** | sourcemap: true, poweredByHeader | ✅ | Reglas YAML (3 reglas) |

### 1.2 Ataques que Celador AÚN NO Detecta

| Tipo de Ataque | Ejemplo Real | Por qué no detecta | Esfuerzo para agregar |
|----------------|-------------|-------------------|----------------------|
| **Ofuscación multi-capa** | base64 dentro de eval dentro de string construido | Solo detecta patrones simples, no cadenas de transformación | Medio — expandir patrones |
| **Protestware** | `colors.js` bucle infinito, `faker.js` corrupto | No ejecuta código ni analiza semántica de loops | Alto — sandbox o análisis de flujo |
| **Confusión de dependencias** | Paquete interno publicado en npm público | No conoce registros privados del usuario | Medio — configuración de namespaces |
| **Paquetes de mantenedor comprometido** | Cuenta npm hackeada | No verifica identidad del publicador | Alto — Sigstore/provenance |
| **Typosquatting de scoped packages** | `@types/expresss` vs `@types/express` | La lista no cubre scoped packages exhaustivamente | Bajo — expandir lista |
| **Malware polimórfico** | Código que se transforma al leer | Análisis estático no puede detectar código que muta | Muy alto — imposible sin sandbox |
| **Supply chain en niveles profundos** | Malware 10 niveles en el árbol | OSV cubre vulns conocidas, no malware nuevo en niveles profundos | Alto — análisis de grafo completo |
| **Archivos binarios en payloads** | `.wasm` malicioso, `.so` troyano | Solo inspecciona `.node` como binario nativo | Medio — expandir detección de binarios |

---

## 2. Brechas Persistentes — Priorizadas

### Críticas (seguridad real)

| Brecha | Impacto | Esfuerzo | Descripción |
|--------|---------|----------|-------------|
| **Procedencia criptográfica** | No puede verificar que el paquete viene del autor legítimo | Alto — integración Sigstore/cosign | Sin esto, un atacante que compromise una cuenta de npm puede publicar paquetes firmados que Celador acepta como legítimos |
| **Análisis de comportamiento (sandbox)** | No detecta protestware ni código que solo se ejecuta en ciertas condiciones | Alto — infraestructura de sandbox | `colors.js` y `faker.js` no tienen patrones estáticos detectables — el código es benigno hasta que se ejecuta con una condición específica |

### Altas (mejora significativa)

| Brecha | Impacto | Esfuerzo | Descripción |
|--------|---------|----------|-------------|
| **Confusión de dependencias** | Vector de ataque enterprise documentado | Medio — config de namespaces privados | Permite que un atacante publique un paquete con nombre que colisiona con un namespace interno de la empresa |
| **Caché granular por paquete** | Cada cambio invalida todo el caché | Medio — refactor de cache keys | Un `npm install express` re-escanea las 500 dependencias existentes, desperdiciando 99.8% del trabajo previo |
| **Ofuscación multi-capa** | Detecta eval() simple pero no eval(atob(base64(x))) | Medio — pattern chaining | Los atacantes sofisticados encadenan 3+ capas de ofuscación que los patrones simples no capturan |

### Medias (mejora incremental)

| Brecha | Impacto | Esfuerzo | Descripción |
|--------|---------|----------|-------------|
| **Scoped packages en typosquatting** | `@scope/pkgs` vs `@scope/pkg` | Bajo — expandir lista | La lista actual no cubre scoped packages populares |
| **Binarios .wasm/.so/.dll** | Malware en WebAssembly o librerías nativas | Medio — detección de archivos binarios | Solo `.node` es verificado actualmente |
| **`fix` re-ejecuta `scan`** | 2x más lento de lo necesario | Bajo — aceptar resultados cacheados | `celador fix` re-ejecuta todo el pipeline de escaneo en vez de reutilizar el último scan |

### Bajas (nice-to-have)

| Brecha | Impacto | Esfuerzo |
|--------|---------|----------|
| Versión check async | 20s timeout si GitHub no responde | Bajo |
| Monorepo support | Un solo root directory | Medio |
| Más ecosistemas | Solo JS/TS/Deno | Alto |
| PRs automatizados | Fix local solamente | Alto |

---

## 3. Mejoras Posibles Faltantes

### 3.1 Seguridad — Mejoras de Alto Impacto

| Mejora | Qué agrega | Valor | Esfuerzo |
|--------|-----------|-------|----------|
| **Verificación de integridad de tarball** | Verificar que el hash del tarball coincide con el del registry | Previene MITM en descarga de paquetes | Bajo — checksum comparison |
| **Detección de paquetes recién publicados** | Alertar sobre paquetes con <7 días de antigüedad | Los ataques de typosquatting suelen ser nuevos | Bajo — metadata de `time.created` |
| **Análisis de grafo de dependencias** | Detectar dependencias innecesarias o sospechosas en el árbol | Identifica paquetes que no deberían estar ahí | Medio — traversal del grafo |
| **Patrones de ofuscación encadenada** | Detectar `eval(atob(...))`, `new Function(decodeURI(...))` | Captura ataques más sofisticados | Medio — regex compuestos |
| **Verificación de maintainer** | Cross-reference con maintainer histórico | Detecta cuentas comprometidas | Medio — npm API metadata |

### 3.2 Rendimiento — Mejoras Posibles

| Mejora | Impacto | Esfuerzo |
|--------|---------|----------|
| Caché por paquete individual | Re-escanea solo deps nuevos, no todos | Medio |
| Reutilizar scan results en fix | `fix` no re-ejecuta `scan` si hay resultado reciente | Bajo |
| Streaming de lockfiles grandes | No carga todo en memoria | Medio |
| Parallel workspace detection | ~9 fs.Stat calls en paralelo | Bajo |

### 3.3 Funcionalidad — Mejoras Posibles

| Mejora | Valor | Esfuerzo |
|--------|-------|----------|
| Modo baseline/diff | "¿qué cambió desde el último escaneo?" | Medio |
| Escaneo incremental | Solo archivos cambiados | Medio |
| Config profiles | Diferentes reglas para dev/prod/CI | Bajo |
| WebAssembly scanning | Detectar .wasm sospechosos | Medio |

---

## 4. Evaluación Honesta de Posicionamiento

### Lo que Celador v0.4.5 es genuinamente:

1. **Un escáner estático de cadena de suministro** — inspecciona lo que el paquete contiene, no solo lo que dice que contiene
2. **Una primera línea de defensa para desarrolladores** — detecta los ataques más comunes antes de que instalen
3. **Un complemento offline-first** — funciona donde Snyk/Socket no pueden (sin internet, sin cuenta)
4. **Un generador de artefactos de cumplimiento** — SBOM + SARIF para CI/CD gates

### Lo que Celador v0.4.5 NO es:

1. **No es zero-trust** — zero-trust requiere procedencia criptográfica + sandbox
2. **No reemplaza a Snyk/Socket.dev** — no tiene análisis de comportamiento ni verificación de maintainer
3. **No es una herramienta enterprise completa** — sin RBAC, audit trails, ni compliance reporting
4. **No detecta todos los ataques** — protestware, ofuscación multi-capa, y confusión de dependencias están fuera de alcance

### Posicionamiento recomendado:

> "Celador es un escáner de seguridad de cadena de suministro para JS/TS/Deno que inspecciona el contenido real de cada paquete — no solo su manifiesto. Detecta código malicioso, typosquatting, y vulnerabilidades conocidas, todo offline después del primer escaneo. Ideal como primera línea de defensa para desarrolladores y gate de seguridad en CI/CD."

---

**Análisis completado:** 14 de abril de 2026  
**Estado del proyecto:** v0.4.5 — 6/6 PRDs implementados  
**Siguiente paso natural:** Procedencia criptográfica (Sigstore) o sandbox de comportamiento
