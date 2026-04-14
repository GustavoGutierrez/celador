package audit

import (
	"encoding/json"
	"strings"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

// topNpmPackagesJSON is an embedded list of well-known npm package names
// used for typosquatting detection.
var topNpmPackagesJSON = []byte(`["react","lodash","express","axios","typescript","chalk","debug","async","commander","moment","request","uuid","webpack","babel","eslint","jest","mocha","underscore","jquery","vue","angular","next","tailwindcss","dotenv","cors","body-parser","morgan","fs-extra","glob","rimraf","semver","yaml","zod","vite","postcss","nodemon","cross-env","ts-node","prettier","react-dom","react-router","react-router-dom","react-redux","redux","prop-types","styled-components","classnames","formik","yup","react-hook-form","material-ui","framer-motion","react-icons","react-select","react-table","react-virtualized","react-window","react-dropzone","react-spring","react-use","react-query","swr","recoil","zustand","mobx","immutable","ramda","date-fns","dayjs","luxon","validator","crypto-js","bcrypt","jsonwebtoken","passport","helmet","compression","cookie-parser","multer","sharp","puppeteer","playwright","cheerio","jsdom","node-fetch","got","superagent","ws","socket.io","redis","ioredis","mysql","pg","sequelize","prisma","mongoose","winston","pino","dotenv","config","joi","celebrate","ajv","type-fest","nanoid","shortid","cuid","slugify","i18next","lodash-es","rambda","typeguards","io-ts","runtypes","superstruct"]`)

// topNpmPackages is the loaded list of well-known npm package names
// used for typosquatting detection.
var topNpmPackages []string

func init() {
	if err := json.Unmarshal(topNpmPackagesJSON, &topNpmPackages); err != nil {
		topNpmPackages = []string{
			"react", "lodash", "express", "axios", "typescript", "chalk",
			"debug", "async", "commander", "moment", "request", "uuid",
			"webpack", "babel", "eslint", "jest", "mocha", "underscore",
			"jquery", "vue", "angular", "next", "tailwindcss", "dotenv",
		}
	}
}

// TyposquatFinding represents a potential typosquatting detection.
type TyposquatFinding struct {
	SuspectedName string
	LikelyTarget  string
	Distance      int
	Severity      shared.Severity
}

// DetectTyposquat checks the given dependencies against a list of
// well-known npm packages and returns findings for any package whose
// name is suspiciously similar (Levenshtein distance <= 2).
func DetectTyposquat(deps []shared.Dependency) []TyposquatFinding {
	if len(topNpmPackages) == 0 {
		return nil
	}

	// Build a set of known packages for fast lookup.
	known := make(map[string]bool, len(topNpmPackages))
	for _, name := range topNpmPackages {
		known[strings.ToLower(name)] = true
	}

	var findings []TyposquatFinding
	for _, dep := range deps {
		name := strings.ToLower(dep.Name)
		if known[name] {
			continue // Exact match — not typosquatting.
		}
		for _, knownName := range topNpmPackages {
			knownLower := strings.ToLower(knownName)
			dist := levenshtein(name, knownLower)
			if dist >= 1 && dist <= 2 {
				severity := shared.SeverityMedium
				if dist == 1 {
					severity = shared.SeverityHigh
				}
				findings = append(findings, TyposquatFinding{
					SuspectedName: dep.Name,
					LikelyTarget:  knownName,
					Distance:      dist,
					Severity:      severity,
				})
				break // One match is enough per dependency.
			}
		}
	}
	return findings
}

// levenshtein computes the Levenshtein distance between two strings.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Use a 2-row matrix for space efficiency.
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for i := 0; i <= len(b); i++ {
		prev[i] = i
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}

	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
