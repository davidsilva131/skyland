/// <reference types="@cloudflare/workers-types" />

interface Env {
  RAILWAY_API_URL?: string;
}

export const onRequest: PagesFunction<Env> = async (context) => {
  const API_URL = context.env.RAILWAY_API_URL;

  if (!API_URL) {
    return new Response(
      JSON.stringify({
        title: 'Proxy misconfigured',
        status: 502,
        detail: 'RAILWAY_API_URL no está configurada.',
        code: 'validation',
      }),
      {
        status: 502,
        headers: { 'Content-Type': 'application/problem+json' },
      }
    );
  }

  const { request, params } = context;
  const path = (params.path as string[] | undefined)?.join('/') ?? '';
  const url = `${API_URL}/api/${path}${new URL(request.url).search}`;

  const headers = new Headers();
  for (const name of ['Cookie', 'Origin', 'Content-Type', 'Accept'] as const) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }

  const body =
    request.method === 'GET' || request.method === 'HEAD' ? undefined : request.body;

  const res = await fetch(url, {
    method: request.method,
    headers,
    body,
    redirect: 'manual',
  });

  const out = new Headers(res.headers);
  // Keep Set-Cookie (the whole point) and content headers; drop hop-by-hop.
  out.delete('content-encoding');
  out.delete('transfer-encoding');
  out.delete('connection');

  return new Response(res.body, { status: res.status, headers: out });
};
