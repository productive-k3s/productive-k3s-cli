# Privacidad Y Telemetría

`pk3s` puede persistir una preferencia local de telemetría, aplicar overrides por comando y propagar contexto anónimo de telemetría hacia las ejecuciones delegadas de Productive K3S Infra o Core.

También puede persistir un bearer token opcional de Observability usado para pedir asociación privada de eventos en el backend de telemetría.

## Objetivos

- mantener la telemetría explícita y auditable
- mantener anónimo el contrato de payload compartido
- permitir que el CLI correlacione una cadena completa de comandos sin filtrar contenido de profiles, IPs ni hostnames

## Orden de resolución

El CLI resuelve la telemetría en este orden:

1. `--telemetry enable|disable` en el comando actual
2. `TELEMETRY_ENABLED` desde el environment
3. la config persistida del CLI en `~/.config/productive-k3s-cli/config.json`
4. un prompt interactivo, con `Yes` como default
5. `false` para corridas no interactivas sin preferencia previa

## Preferencia persistida

El CLI expone comandos explícitos de configuración:

```bash
pk3s config telemetry enable
pk3s config telemetry disable
pk3s config telemetry status
pk3s config observability set pk3s_live_xxxxx
pk3s config observability clear
pk3s config observability status
```

Esos comandos afectan corridas futuras. Un override con `--telemetry ...` sigue mandando para la invocación actual.

El token de Observability es independiente del toggle de consentimiento de telemetría:

- si la telemetría está deshabilitada, no se envía nada aunque exista un token
- si la telemetría está habilitada y no existe token, los eventos siguen siendo anónimos
- si la telemetría está habilitada y existe token, el CLI sigue enviando el marker público y además `Authorization: Bearer ...`

## Contrato de propagación

Cuando la telemetría está habilitada, el CLI envía sus propios eventos y propaga contexto hacia el bundle delegado:

- `TELEMETRY_ENABLED`
- `TELEMETRY_ENDPOINT`
- `TELEMETRY_MARKER`
- `TELEMETRY_BEARER_TOKEN`
- `TELEMETRY_MAX_RETRIES`
- `TELEMETRY_CONNECT_TIMEOUT_SECONDS`
- `TELEMETRY_REQUEST_TIMEOUT_SECONDS`
- `TELEMETRY_OUTBOX_DIR`
- `TELEMETRY_USER_AGENT`
- `TELEMETRY_SESSION_ID`
- `TELEMETRY_PARENT_RUN_ID`
- `TELEMETRY_COMPONENT`

La capa delegada genera después su propio `run_id` y conserva la cadena de correlación compartida.

Endpoint por default: `https://telemetry.productive-k3s.io/telemetry`
Header marker por default: `X-Productive-K3S-Telemetry: pk3s-public-v1`
Header privado opcional: `Authorization: Bearer <telemetry-token>`

## Modelo de correlación

- `session_id`: compartido por toda la operación lógica
- `run_id`: único para la ejecución del componente actual
- `parent_run_id`: ejecución padre directa dentro de la cadena

Eso permite reconstruir flujos como `pk3s apply -> scenario de infra -> bootstrap de core`.

## Contrato de privacidad

El payload de telemetría del CLI está pensado para incluir:

- nombre del comando
- componente target (`infra` o `core`)
- plataforma y arquitectura local
- si se usó o no un flujo basado en profile
- metadata anónima de correlación

No está pensado para incluir:

- contenido del archivo de profile
- direcciones IP
- hostnames
- usernames
- directorios de trabajo locales

La telemetría sigue siendo best-effort y nunca debe romper la ejecución del CLI.
