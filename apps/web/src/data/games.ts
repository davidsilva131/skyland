export type GameStatus = 'live' | 'soon';

export interface Game {
  slug: string;
  name: string;
  emoji: string;
  description: string;
  status: GameStatus;
}

export const games: Game[] = [
  {
    slug: 'animalitos',
    name: 'Animalitos',
    emoji: '🐓',
    description: 'Sorteos rápidos con pagos fijos. ¡El clásico venezolano!',
    status: 'soon',
  },
  {
    slug: 'caballos',
    name: 'Caballos',
    emoji: '🐎',
    description: 'La emoción de las carreras. Próximamente.',
    status: 'soon',
  },
];

export function statusLabel(status: GameStatus): string {
  return status === 'live' ? 'Disponible' : 'Próximamente';
}
