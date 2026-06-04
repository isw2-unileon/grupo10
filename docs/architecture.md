.
├── backend/                     # Backend en Go
│   ├── cmd/
│   │   └── server/
│   │       └── main.go          # Composition Root: ensamblaje e inyección de dependencias
│   ├── internal/                # Núcleo de la aplicación (protegido de importaciones externas)
│   │   ├── users/               # Módulo de usuarios y perfiles
│   │   ├── notes/               # Módulo del "Pipeline de Revisión" de apuntes
│   │   ├── calendar/            # Módulo de gestión de tiempo y tutorías
│   │   └── analytics/           # Módulo de estadísticas y rendimiento
│   └── migrations/              # Esquema de BD (up.sql / down.sql)
├── frontend/                    # SPA con Vue 3 + Vite + TypeScript (Vue Router + Pinia)
├── e2e/                         # Tests end-to-end (Playwright)
├── docs/                        # Documentación técnica, modelos y decisiones (ADRs)
├── .github/                     # Workflows para CI/CD (GitHub Actions)
├── go.mod                       # Definición del módulo de Go (raíz)
└── README.md                    # Guía para desarrolladores 