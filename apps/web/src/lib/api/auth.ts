// Typed auth client over the generated contract (types.gen.ts).
// Every call is same-origin (`/api/v1/...`) — the vite dev proxy and the
// Cloudflare Pages Function both forward `/api/*` to the Go API, so the
// SameSite=Lax cookie rides along (spec §5 "Same-origin API").
import type { components, operations } from './types.gen';

export type Player = components['schemas']['Player'];
export type ProblemCode = components['schemas']['Problem']['code'];
export type LoginInput = operations['login']['requestBody']['content']['application/json'];
export type RegisterInput = operations['register']['requestBody']['content']['application/json'];

export class AuthError extends Error {
  code: ProblemCode | 'network';
  detail: string;

  constructor(code: ProblemCode | 'network', detail: string) {
    super(detail);
    this.code = code;
    this.detail = detail;
  }
}

const BASE = '/api/v1';

/** problem+json → AuthError; non-JSON or network failure → code 'network'. */
async function parseProblem(res: Response): Promise<AuthError> {
  try {
    const body = (await res.json()) as components['schemas']['Problem'];
    if (body && typeof body.code === 'string') {
      return new AuthError(body.code, body.detail);
    }
  } catch {
    // fall through: not problem+json
  }
  return new AuthError('network', 'No se pudo conectar con el servidor. Intenta más tarde.');
}

async function post(path: '/auth/login' | '/auth/register' | '/auth/logout', body?: unknown) {
  let res: Response;
  try {
    res = await fetch(`${BASE}${path}`, {
      method: 'POST',
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: 'include',
    });
  } catch {
    throw new AuthError('network', 'No se pudo conectar con el servidor. Intenta más tarde.');
  }
  if (!res.ok) throw await parseProblem(res);
  return res;
}

export async function login(input: LoginInput): Promise<Player> {
  const res = await post('/auth/login', input);
  return (await res.json()) as Player;
}

export async function register(input: RegisterInput): Promise<Player> {
  const res = await post('/auth/register', input);
  return (await res.json()) as Player;
}

export async function logout(): Promise<void> {
  await post('/auth/logout');
}

export type MeResult = { status: 'ok'; player: Player } | { status: 'unauthenticated' };

/** `/me` — 401 is an expected outcome, surfaced to the caller (spec §5). */
export async function me(): Promise<MeResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/auth/me`, { credentials: 'include' });
  } catch {
    throw new AuthError('network', 'No se pudo conectar con el servidor. Intenta más tarde.');
  }
  if (res.status === 401) return { status: 'unauthenticated' };
  if (!res.ok) throw await parseProblem(res);
  return { status: 'ok', player: (await res.json()) as Player };
}
