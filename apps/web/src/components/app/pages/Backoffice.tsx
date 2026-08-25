export default function Backoffice() {
  return (
    <div>
      <h1 className="text-2xl font-black tracking-tight">Backoffice</h1>
      <p className="mt-1 text-sm text-zinc-400">Gestión interna (Admin / Soporte). Protegido por RBAC.</p>

      <div className="mt-6 grid gap-4 md:grid-cols-2">
        <div className="rounded-2xl border border-white/10 bg-zinc-900 p-5">
          <p className="text-2xl">💳</p>
          <h2 className="mt-2 font-bold text-white">Recargas</h2>
          <p className="mt-1 text-sm text-zinc-400">
            Verificar referencias de pago móvil y acreditar saldo (PagoVerifier).
          </p>
        </div>
        <div className="rounded-2xl border border-white/10 bg-zinc-900 p-5">
          <p className="text-2xl">👥</p>
          <h2 className="mt-2 font-bold text-white">Jugadores</h2>
          <p className="mt-1 text-sm text-zinc-400">
            Ver cuentas, saldos y movimientos del ledger.
          </p>
        </div>
        <div className="rounded-2xl border border-white/10 bg-zinc-900 p-5">
          <p className="text-2xl">🎮</p>
          <h2 className="mt-2 font-bold text-white">Juegos</h2>
          <p className="mt-1 text-sm text-zinc-400">
            Configurar catálogo, payouts y límites por juego.
          </p>
        </div>
        <div className="rounded-2xl border border-white/10 bg-zinc-900 p-5">
          <p className="text-2xl">🧾</p>
          <h2 className="mt-2 font-bold text-white">Ledger</h2>
          <p className="mt-1 text-sm text-zinc-400">
            Auditoría de cada movimiento de saldo (doble entrada).
          </p>
        </div>
      </div>
    </div>
  );
}