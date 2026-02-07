// Главный компонент приложения с маршрутизацией
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';

// Импортируем страницы
import Login from './pages/Login';
import Register from './pages/Register';
import Dashboard from './pages/Dashboard';
import CreateDelivery from './pages/CreateDelivery';
import TrackDelivery from './pages/TrackDelivery';

// Импортируем стили
import './styles/global.css';

// Компонент для защиты приватных маршрутов
// Если пользователь не залогинен - перенаправляем на /login
function PrivateRoute({ children }) {
  const token = localStorage.getItem('token');
  return token ? children : <Navigate to="/login" />;
}

// Компонент для публичных маршрутов (вход/регистрация)
// Если пользователь уже залогинен - перенаправляем на дашборд
function PublicRoute({ children }) {
  const token = localStorage.getItem('token');
  return token ? <Navigate to="/dashboard" /> : children;
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Главная страница - редирект */}
        <Route 
          path="/" 
          element={<Navigate to="/dashboard" />} 
        />

        {/* Публичные маршруты (доступны только незалогиненным) */}
        <Route 
          path="/login" 
          element={
            <PublicRoute>
              <Login />
            </PublicRoute>
          } 
        />
        <Route 
          path="/register" 
          element={
            <PublicRoute>
              <Register />
            </PublicRoute>
          } 
        />

        {/* Приватные маршруты (требуют авторизации) */}
        <Route 
          path="/dashboard" 
          element={
            <PrivateRoute>
              <Dashboard />
            </PrivateRoute>
          } 
        />
        <Route 
          path="/create-delivery" 
          element={
            <PrivateRoute>
              <CreateDelivery />
            </PrivateRoute>
          } 
        />
        <Route 
          path="/track/:id" 
          element={
            <PrivateRoute>
              <TrackDelivery />
            </PrivateRoute>
          } 
        />

        {/* 404 страница */}
        <Route 
          path="*" 
          element={
            <div style={{ 
              minHeight: '100vh', 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              flexDirection: 'column',
              gap: '1rem'
            }}>
              <h1 style={{ fontSize: '4rem' }}>404</h1>
              <p>Страница не найдена</p>
              <a href="/dashboard" className="btn btn-primary">
                На главную
              </a>
            </div>
          } 
        />
      </Routes>
    </BrowserRouter>
  );
}

export default App;