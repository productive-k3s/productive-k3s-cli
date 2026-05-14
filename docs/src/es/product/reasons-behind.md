# Razones Del Diseño

El CLI existe para resolver tres problemas concretos:

## Una sola superficie de comandos

Los usuarios no deberían tener que memorizar entrypoints separados para Core e Infra. `pk3s` les da un solo ejecutable y un solo sistema de ayuda.

## Ejecución consciente de releases

El CLI puede resolver bundles publicados en GitHub Releases y mantener alineadas las versiones de Core e Infra mediante un manifest explícito.

## Comportamiento remote-first más seguro

Para quienes consumen el proyecto como producto, el modo remoto es el default porque valida exactamente los artifacts publicados que luego descargará el resto.

El modo local sigue existiendo, pero únicamente como elección explícita de desarrollo.
