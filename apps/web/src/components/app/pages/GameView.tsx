import { Link, useParams } from 'react-router-dom';

export default function GameView() {
  const { slug } = useParams<{ slug: string }>();

  return (
    <div>
      <Link to="/" className="text-sm font-semibold text-zinc-500 hover:text-zinc-300">
        ← Lobby
      </Link>
      <h1 className="mt-2 text-2xl font-black tracking-tight capitalize">{slug}</h1>

      <div className="mt-6 rounded-2xl border border-white/10 bg-zinc-900 p-8 text-center">
        <p className="text-4xl">🚧</p>
        <p className="mt-3 font-semibold text-white">
          El motor de juegos está en construcción
        </p>
        <p className="mx-auto mt-1 max-w-sm text-sm text-zinc-400">
          Aquí vivirán el tablero de apuestas, el contador de sorteo en vivo (SSE) y tus
          jugadas en cuanto definamos la mecánica de los animalitos.
        </p>
      </div>
    </div>
  );
}