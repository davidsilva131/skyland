import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { SessionProvider, useSession } from './session';
import AppLayout from './AppLayout';
import Lobby from './pages/Lobby';
import GameView from './pages/GameView';
import Backoffice from './pages/Backoffice';

/**
 * SPA de Skyland (montada en /app por Astro).
 * Rutas:
 *  /app            → Lobby (catálogo de juegos)
 *  /app/play/:slug → Vista de un juego
 *  /app/backoffice → Panel admin/soporte
 */

/** Gate (ADR-0004): nothing player-facing mounts before /me resolves. */
function Gate({ children }: { children: React.ReactNode }) {
  const { loading } = useSession();
  if (loading) {
    // zinc-950 splash — no Lobby flash; the redirect runs at document level.
    return (
      <div className="grid min-h-dvh place-items-center bg-zinc-950">
        <span className="animate-pulse text-2xl font-black tracking-tight text-amber-400">
          🎲 Skyland
        </span>
      </div>
    );
  }
  return <>{children}</>;
}

export default function App() {
  return (
    <BrowserRouter basename="/app">
      <SessionProvider>
        <Gate>
          <Routes>
            <Route element={<AppLayout />}>
              <Route index element={<Lobby />} />
              <Route path="play/:slug" element={<GameView />} />
              <Route path="backoffice" element={<Backoffice />} />
            </Route>
          </Routes>
        </Gate>
      </SessionProvider>
    </BrowserRouter>
  );
}