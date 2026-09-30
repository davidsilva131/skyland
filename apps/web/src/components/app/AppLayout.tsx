import { useEffect, useState } from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import { useSession } from './session';

const nav = [
  { to: '/', label: 'Lobby', icon: '🎰' },
  { to: '/backoffice', label: 'Admin', icon: '🛠️' },
];

/** Saldo desde la API de wallets (mismo origen, cookie de sesión). */
function Saldo() {
  // ponytail: un fetch sin caché ni revalidación; se mueve a un store
  // compartido cuando la primera segunda vista necesite el saldo.
  const [state, setState] = useState<
    { kind: 'loading' } | { kind: 'error' } | { kind: 'ok'; monto: number; moneda: string }
  >({ kind: 'loading' });

  useEffect(() => {
    let cancelled = false;
    fetch('/api/v1/wallets/saldo', { credentials: 'include' })
      .then(async (res) => {
        if (!res.ok) throw new Error(`saldo ${res.status}`);
        const body = (await res.json()) as { monto: number; moneda: string };
        if (!cancelled) setState({ kind: 'ok', monto: body.monto, moneda: body.moneda });
      })
      .catch(() => {
        if (!cancelled) setState({ kind: 'error' });
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (state.kind === 'loading') {
    return (
      <span className="rounded-lg border border-white/10 px-3 py-1.5 text-sm font-semibold text-zinc-500">
        Saldo: …
      </span>
    );
  }
  if (state.kind === 'error') {
    return (
      <span
        className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-1.5 text-sm font-semibold text-red-400"
        title="No se pudo cargar el saldo. Intenta recargar la página."
      >
        Saldo: —
      </span>
    );
  }
  // El monto llega en la mínima unidad (céntimos) — se muestra en bolívares.
  const formatted = new Intl.NumberFormat('es-VE', {
    style: 'currency',
    currency: 'VES',
  }).format(state.monto / 100);
  return (
    <span className="rounded-lg border border-white/10 px-3 py-1.5 text-sm font-semibold text-zinc-300">
      Saldo: <span className="text-amber-400">{formatted}</span>
    </span>
  );
}

export default function AppLayout() {
  const { player, logout } = useSession();

  return (
    <div className="min-h-dvh bg-zinc-950 text-zinc-100 pb-20 md:pb-0">
      {/* Header */}
      <header className="sticky top-0 z-20 border-b border-white/10 bg-zinc-950/90 backdrop-blur">
        <div className="flex items-center justify-between px-4 py-3">
          <NavLink to="/" className="flex items-center gap-2 font-black tracking-tight text-amber-400">
            <span className="grid size-8 place-items-center rounded-lg bg-amber-400/10">🎲</span>
            Skyland
          </NavLink>
          <div className="flex items-center gap-3">
            <Saldo />
            {player && (
              <div className="flex items-center gap-2">
                <span className="hidden text-sm text-zinc-400 sm:inline">{player.email}</span>
                <button
                  type="button"
                  onClick={() => void logout()}
                  className="rounded-lg border border-white/10 px-3 py-1.5 text-sm font-semibold text-zinc-300 transition hover:border-amber-400/50 hover:text-white"
                >
                  Salir
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Contenido */}
      <main className="mx-auto max-w-3xl px-4 py-6">
        <Outlet />
      </main>

      {/* Bottom nav (móvil) */}
      <nav className="fixed inset-x-0 bottom-0 z-20 border-t border-white/10 bg-zinc-950/95 backdrop-blur md:hidden">
        <div className="flex">
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                `flex flex-1 flex-col items-center gap-0.5 py-2.5 text-xs font-semibold transition ${
                  isActive ? 'text-amber-400' : 'text-zinc-500 hover:text-zinc-300'
                }`
              }
            >
              <span className="text-lg">{item.icon}</span>
              {item.label}
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  );
}