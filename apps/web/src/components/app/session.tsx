// Session state for the /app SPA (spec §5): /me on mount, full-page
// redirect to /login when unauthenticated — /login is an Astro page outside
// the /app basename, so a router navigate can't reach it (spec §5, ADR-0004).
import { createContext, useContext, useEffect, useState } from 'react';
import { me, logout as apiLogout, type Player } from '../../lib/api/auth';

type SessionState =
  | { phase: 'loading' }
  | { phase: 'redirecting' }
  | { phase: 'ready'; player: Player };

interface Session {
  player: Player | null;
  loading: boolean;
  logout: () => Promise<void>;
}

const SessionContext = createContext<Session | null>(null);

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<SessionState>({ phase: 'loading' });

  useEffect(() => {
    let cancelled = false;
    me()
      .then((res) => {
        if (cancelled) return;
        if (res.status === 'ok') {
          setState({ phase: 'ready', player: res.player });
        } else {
          // Full-page redirect: the cookie gate lives at the document level.
          setState({ phase: 'redirecting' });
          window.location.href = '/login';
        }
      })
      .catch(() => {
        if (cancelled) return;
        setState({ phase: 'redirecting' });
        window.location.href = '/login';
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function logout() {
    await apiLogout();
    window.location.href = '/login';
  }

  const value =
    state.phase === 'ready'
      ? { player: state.player, loading: false, logout }
      : { player: null, loading: true, logout };

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): Session {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error('useSession must be used within SessionProvider');
  return ctx;
}
