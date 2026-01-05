# Lumina Development Tiltfile

# Load Docker Compose
docker_compose('deployments/docker/docker-compose.yml')

# Gateway (Go backend) - rebuild image on changes (no live update to avoid constant syncing)
docker_build(
    'lumina-gateway',
    context='apps/gateway',
    dockerfile='apps/gateway/Dockerfile.dev',
    only=['./cmd/', './internal/', './go.mod', './go.sum', './.air.toml'],
    ignore=['**/tmp', '**/tmp/**', '../web', '**/node_modules', '**/.next', '**/*.exe', '**/build-errors.log', '**/*~', '**/.#*', '**/#*#']
)

# Web (Next.js frontend) with hot reload
docker_build(
    'lumina-web',
    context='apps/web',
    dockerfile='apps/web/Dockerfile.dev',
    only=['./src/', './public/', './package.json', './next.config.js', './tailwind.config.ts', './tsconfig.json', './postcss.config.js', './components.json'],
    ignore=['../gateway', '**/node_modules', '**/.next', '**/out', '**/.turbo', '**/*.tsbuildinfo']
)

# Docker Compose resource configurations
dc_resource('gateway', labels=['app'], resource_deps=['postgres', 'redis', 'opensearch'])
dc_resource('web', labels=['app'], resource_deps=['gateway'])
dc_resource('postgres', labels=['infra'])
dc_resource('redis', labels=['infra'])
dc_resource('opensearch', labels=['infra'])
dc_resource('opensearch-dashboards', labels=['infra'], resource_deps=['opensearch'])

# Local resources for running commands
local_resource(
    'go-test',
    cmd='cd apps/gateway && go test -v ./...',
    labels=['test'],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL
)

# Display helpful information
print("""
╔══════════════════════════════════════════════════════════════╗
║                     Lumina Development                       ║
╠══════════════════════════════════════════════════════════════╣
║  Gateway API:          http://localhost:8080                 ║
║  Web Dashboard:        http://localhost:3000                 ║
║  OpenSearch:           http://localhost:9200                 ║
║  OpenSearch Dashboards: http://localhost:5601                ║
║  PostgreSQL:           localhost:5432                        ║
║  Redis:                localhost:6379                        ║
╚══════════════════════════════════════════════════════════════╝
""")
