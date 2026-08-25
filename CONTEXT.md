# Skyland — Ubiquitous Language (Glosario)

> Glosario del dominio. Solo términos y su definición canónica — sin detalles de implementación.

## Términos

| Término | Definición canónica |
|---|---|
| **Jugador** | Usuario registrado con saldo que participa en los juegos de la plataforma. |
| **Casa** | Operador de la plataforma (Skyland). Contraparte de cada apuesta. |
| **Saldo** | Dinero disponible del jugador para apostar. Hoy en **VES (bolívares)**; el diseño es **multi-moneda** desde el día 1. |
| **Recarga** | Depósito de fondos realizado por el jugador para aumentar su saldo. Vía **pago móvil** (Venezuela). |
| **Pago móvil** | Método de pago interbancario venezolano: el jugador transfiere monto a la cuenta de la Casa y reporta referencia de 6 dígitos. Verificación vía API de conciliación bancaria (BDV/Mercantil) o manual. |
| **Retiro** | Movimiento de fondos del saldo del jugador hacia fuera de la plataforma. **Fuera del MVP**; se diseñará el modelo desde el inicio. |
| **Juego** | Unidad del catálogo de la plataforma (animalitos, caballos, futuros). Cada juego define sus propias reglas, mercado de apuestas y fuente de resultados. |
| **Animalitos** | Juego de lotería de animales. Versión propia de la Casa (resultados por RNG de la plataforma, no contra lotería oficial). |
| **Caballos** | Juego de carreras de caballos (futuro). |
| **Apuesta** | Compromiso de una cantidad de saldo del jugador sobre un resultado posible en un juego. |
| **Resultado** | Desenlace de una ronda/sorteo de un juego que determina apuestas ganadoras y perdedoras. |
| **Sorteo / Ronda** | Evento periódico de un juego en el que se generan resultados. |
| **Payout** | Multiplicador que determina la ganancia pagada por una apuesta ganadora (incluye la devolución del monto apostado o no según el tipo de jugada). |
| **RNG** | Generador de números aleatorios de la plataforma, fuente de resultados de los juegos. |
| **Mercado de apuesta** | Tipo de jugada disponible dentro de un juego (ej: animal suelto, caballo — combinación de animales). |
| **Concurrente** | Usuarios conectados/apostando simultáneamente (objetivo: 100–1.000 con picos en horarios de sorteo). |
| **Credencial bancaria** | Claves de API de conciliación (BDVenLínea Empresa / Mercantil) que permiten verificar pagos móviles automáticamente. |