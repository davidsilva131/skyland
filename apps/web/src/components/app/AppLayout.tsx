import { NavLink, Outlet } from 'react-router-dom';

const nav = [
  { to: '/', label: 'Lobby', icon: '🎰' },
  { to: '/backoffice', label: 'Admin', icon: '🛠️' },
];

export default function AppLayout() {
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
            <span className="rounded-lg border border-white/10 px-3 py-1.5 text-sm font-semibold text-zinc-300">
              Saldo: <span className="text-amber-400">Bs 0,00</span>
            </span>
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