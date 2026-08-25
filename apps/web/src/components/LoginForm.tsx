import { useState } from 'react';

type Mode = 'login' | 'register';

const API_BASE = import.meta.env.PUBLIC_API_URL ?? '/api';

export default function LoginForm() {
  const [mode, setMode] = useState<Mode>(() =>
    typeof window !== 'undefined' &&
    new URLSearchParams(window.location.search).get('mode') === 'register'
      ? 'register'
      : 'login'
  );
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [birthdate, setBirthdate] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/auth/${mode === 'login' ? 'login' : 'register'}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(
          mode === 'register' ? { email, password, birthdate } : { email, password }
        ),
        credentials: 'include',
      });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        setError(body?.detail ?? 'Algo salió mal. Intenta de nuevo.');
        return;
      }
      window.location.href = '/app';
    } catch {
      setError('No se pudo conectar con el servidor. Intenta más tarde.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <div className="flex rounded-xl bg-zinc-800 p-1 text-sm font-semibold">
        {(['login', 'register'] as const).map((m) => (
          <button
            key={m}
            type="button"
            onClick={() => setMode(m)}
            className={`flex-1 rounded-lg px-4 py-2 transition ${
              mode === m ? 'bg-amber-400 text-zinc-950' : 'text-zinc-400 hover:text-white'
            }`}
          >
            {m === 'login' ? 'Entrar' : 'Crear cuenta'}
          </button>
        ))}
      </div>

      <label className="block">
        <span className="mb-1 block text-sm font-medium text-zinc-300">Correo electrónico</span>
        <input
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="w-full rounded-xl border border-white/10 bg-zinc-800 px-4 py-2.5 text-white outline-none focus:border-amber-400"
          placeholder="tucorreo@ejemplo.com"
        />
      </label>

      <label className="block">
        <span className="mb-1 block text-sm font-medium text-zinc-300">Contraseña</span>
        <input
          type="password"
          required
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="w-full rounded-xl border border-white/10 bg-zinc-800 px-4 py-2.5 text-white outline-none focus:border-amber-400"
          placeholder="••••••••"
        />
      </label>

      {mode === 'register' && (
        <label className="block">
          <span className="mb-1 block text-sm font-medium text-zinc-300">Fecha de nacimiento (+18)</span>
          <input
            type="date"
            required
            value={birthdate}
            onChange={(e) => setBirthdate(e.target.value)}
            className="w-full rounded-xl border border-white/10 bg-zinc-800 px-4 py-2.5 text-white outline-none focus:border-amber-400"
          />
        </label>
      )}

      {error && (
        <p className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
          {error}
        </p>
      )}

      <button
        type="submit"
        disabled={loading}
        className="w-full rounded-xl bg-amber-400 px-4 py-3 font-bold text-zinc-950 transition hover:bg-amber-300 disabled:opacity-50"
      >
        {loading ? 'Un momento…' : mode === 'login' ? 'Entrar' : 'Crear cuenta y jugar'}
      </button>
    </form>
  );
}