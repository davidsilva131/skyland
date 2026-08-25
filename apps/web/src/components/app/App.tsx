import { BrowserRouter, Route, Routes } from 'react-router-dom';
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
export default function App() {
  return (
    <BrowserRouter basename="/app">
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<Lobby />} />
          <Route path="play/:slug" element={<GameView />} />
          <Route path="backoffice" element={<Backoffice />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}