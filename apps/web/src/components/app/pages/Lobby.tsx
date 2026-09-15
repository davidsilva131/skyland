import { Link } from 'react-router-dom';
import { games, statusLabel } from '../../../data/games';

export default function Lobby() {
  return (
    <div>
      <h1 className="text-2xl font-black tracking-tight">Lobby</h1>
      <p className="mt-1 text-sm text-zinc-400">Elige un juego y apuesta con tu saldo.</p>

      <div className="mt-6 grid gap-4">
        {games.map((game) => (
          <Link
            key={game.slug}
            to={`/play/${game.slug}`}
            className="flex items-center gap-4 rounded-2xl border border-white/10 bg-zinc-900 p-4 transition hover:border-amber-400/50"
          >
            <span className="grid size-14 place-items-center rounded-xl bg-zinc-800 text-3xl">
              {game.emoji}
            </span>
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <h2 className="font-bold text-white">{game.name}</h2>
                <span className="rounded-full bg-amber-400/10 px-2 py-0.5 text-[10px] font-bold uppercase text-amber-400">
                  {statusLabel(game.status)}
                </span>
              </div>
              <p className="mt-0.5 text-sm text-zinc-400">{game.description}</p>
            </div>
            <span className="text-zinc-600">→</span>
          </Link>
        ))}
      </div>
    </div>
  );
}