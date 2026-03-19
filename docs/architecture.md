.
├── cmd/                         # Puntos de entrada de la aplicación 
│   └── api/
│       └── main.go              # Composition Root: Ensamblaje e inyección de dependencias
├── internal/                    # Núcleo de la aplicación (protegido de importaciones externas)
│   ├── users/                   # Módulo de usuarios y perfiles 
│   ├── notes/                   # Módulo del "Pipeline de Revisión" de apuntes 
│   ├── calendar/                # Módulo de gestión de tiempo y tutorías 
│   └── analytics/               # Módulo de estadísticas y rendimiento 
├── docs/                        # Documentación técnica, modelos y decisiones 
├── web/                         # Frontend (React/Next.js/etc.) 
├── api/                         # Definiciones de contratos (OpenAPI/Swagger)
├── .github/                     # Workflows para CI/CD (GitHub Actions) 
├── go.mod                       # Definición de módulos de Go
└── README.md                    # Guía para desarrolladores 