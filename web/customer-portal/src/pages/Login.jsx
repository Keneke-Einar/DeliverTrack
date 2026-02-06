// Страница входа в систему
import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { login } from '../services/api';
import '../styles/auth.css';

function Login() {
  // Хук для перенаправления на другие страницы
  const navigate = useNavigate();
  
  // Состояния для полей формы
  const [identifier, setIdentifier] = useState(''); // username или email
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // Обработчик отправки формы
  const handleSubmit = async (e) => {
    e.preventDefault(); // Отменяем перезагрузку страницы
    setError(''); // Очищаем предыдущие ошибки
    setLoading(true); // Показываем загрузку

    try {
      // Отправляем запрос на бэкенд
      const data = await login(identifier, password);
      
      // Если успешно - перенаправляем на главную
      navigate('/dashboard');
    } catch (err) {
      // Если ошибка - показываем сообщение
      setError(
        err.response?.data?.error || 
        'Неверный логин или пароль'
      );
    } finally {
      setLoading(false); // Убираем загрузку
    }
  };

  return (
    <div className="auth-container">
      <div className="auth-card fade-in">
        {/* Логотип */}
        <div className="auth-header">
          <div className="logo">
            <span className="logo-icon">📦</span>
            <h1>DeliverTrack</h1>
          </div>
          <p className="auth-subtitle">Войдите в свой аккаунт</p>
        </div>

        {/* Форма входа */}
        <form onSubmit={handleSubmit} className="auth-form">
          {/* Показываем ошибку если есть */}
          {error && (
            <div className="error-message">
              {error}
            </div>
          )}

          {/* Поле логина */}
          <div className="form-group">
            <label htmlFor="identifier">Логин или Email</label>
            <input
              id="identifier"
              type="text"
              placeholder="Введите логин или email"
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              required
              autoFocus
            />
          </div>

          {/* Поле пароля */}
          <div className="form-group">
            <label htmlFor="password">Пароль</label>
            <input
              id="password"
              type="password"
              placeholder="Введите пароль"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>

          {/* Кнопка входа */}
          <button 
            type="submit" 
            className="btn btn-primary btn-lg"
            disabled={loading}
          >
            {loading ? (
              <>
                <span className="loading"></span>
                Вход...
              </>
            ) : (
              'Войти'
            )}
          </button>

          {/* Ссылка на регистрацию */}
          <div className="auth-footer">
            <p className="text-muted">
              Нет аккаунта?{' '}
              <Link to="/register" className="auth-link">
                Зарегистрироваться
              </Link>
            </p>
          </div>
        </form>
      </div>
    </div>
  );
}

export default Login;