// LoginForm (spec §5): login/register modes (?mode=register), visibility
// toggle, terms checkbox, error copy per the code table. States:
// idle / loading (submit disabled) / error / success → /app.
import { useState } from 'react';
import { login, register, AuthError, type ProblemCode } from '../lib/api/auth';

type Mode = 'login' | 'register';

// Error copy (spec §5): code → Spanish; `validation` shows the server
// detail verbatim; anything else (network/unknown) gets the generic line.
const ERROR_COPY: Record<Exclude<ProblemCode, 'validation'>, string> = {
  email_taken: 'Ese correo ya tiene una cuenta. Entra con tu contraseña.',
  invalid_credentials: 'Correo o contraseña incorrectos.',
  underage: 'Debes tener 18 años o más para crear una cuenta.',
  terms_not_accepted: 'Debes aceptar los términos para continuar.',
  rate_limited: 'Demasiados intentos. Espera un momento y prueba de nuevo.',
  unauthenticated: 'Tu sesión expiró. Entra de nuevo.',
  origin_rejected: 'Petición no permitida desde este origen.',
};

const NETWORK_COPY = 'No se pudo conectar con el servidor. Intenta más tarde.';

function errorCopy(err: AuthError): string {
  if (err.code === 'validation') return err.detail;
  return (ERROR_COPY as Record<string, string>)[err.code] ?? NETWORK_COPY;
}

const inputClass =
  'w-full rounded-xl border border-white/10 bg-zinc-800 px-4 py-2.5 text-white outline-none focus:border-amber-400';

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
  const [acceptsTerms, setAcceptsTerms] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function switchMode(m: Mode) {
    setMode(m);
    setError(null);
    // Keep ?mode=register in the URL so a reload stays on the same mode.
    if (typeof window !== 'undefined') {
      const url = new URL(window.location.href);
      if (m === 'register') url.searchParams.set('mode', 'register');
      else url.searchParams.delete('mode');
      window.history.replaceState(null, '', url);
    }
  }

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      if (mode === 'login') {
        await login({ email, password });
      } else {
        // acceptsTerms is `true` on the wire (enum in the schema) — the
        // unchecked box is stopped by the required checkbox before submit.
        await register({ email, password, birthdate, acceptsTerms: true });
      }
      // Register logs you in (201 sets the cookie) — no follow-up login call.
      window.location.href = '/app';
    } catch (err) {
      setError(errorCopy(err instanceof AuthError ? err : new AuthError('network', '')));
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
            onClick={() => switchMode(m)}
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
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className={inputClass}
          placeholder="tucorreo@ejemplo.com"
        />
      </label>

      <label className="block">
        <span className="mb-1 block text-sm font-medium text-zinc-300">Contraseña</span>
        <span className="relative block">
          <input
            type={showPassword ? 'text' : 'password'}
            required
            minLength={8}
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className={`${inputClass} pr-11`}
            placeholder="••••••••"
          />
          <button
            type="button"
            aria-label={showPassword ? 'Ocultar contraseña' : 'Mostrar contraseña'}
            onClick={() => setShowPassword((v) => !v)}
            className="absolute inset-y-0 right-0 grid w-10 place-items-center text-zinc-400 hover:text-white"
          >
            {showPassword ? '🙈' : '👁️'}
          </button>
        </span>
      </label>

      {mode === 'register' && (
        <>
          <label className="block">
            <span className="mb-1 block text-sm font-medium text-zinc-300">
              Fecha de nacimiento (+18)
            </span>
            <input
              type="date"
              required
              autoComplete="bday"
              value={birthdate}
              onChange={(e) => setBirthdate(e.target.value)}
              className={inputClass}
            />
          </label>

          <label className="flex items-start gap-2 text-sm text-zinc-300">
            <input
              type="checkbox"
              required
              checked={acceptsTerms}
              onChange={(e) => setAcceptsTerms(e.target.checked)}
              className="mt-0.5 size-4 accent-amber-400"
            />
            <span>
              Acepto los{' '}
              <a
                href="/terminos"
                target="_blank"
                rel="noopener noreferrer"
                className="text-amber-400 underline underline-offset-2 hover:text-amber-300"
              >
                términos y condiciones
              </a>
            </span>
          </label>
        </>
      )}

      {error && (
        <p
          role="alert"
          className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400"
        >
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
