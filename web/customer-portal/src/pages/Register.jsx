// Страница регистрации
import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { register } from '../services/api';
import '../styles/auth.css';

function Register() {
  const navigate = useNavigate();
  
  // Состояния формы
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    role: 'customer' // По умолчанию регистрируем как клиента
  });
  
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // Обработчик изменения полей
  const handleChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value
    });
  };

  // Обработчик отправки формы
  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    // Проверка паролей
    if (formData.password !== formData.confirmPassword) {
      setError('Пароли не совпадают');
      return;
    }

    // Проверка длины пароля
    if (formData.password.length < 6) {
      setError('Пароль должен быть не менее 6 символов');
      return;
    }

    setLoading(true);

    try {
      // Отправляем запрос на регистрацию
      await register(
        formData.username,
        formData.email,
        formData.password,
        formData.role
      );
      
      // Перенаправляем на страницу входа
      navigate('/login', { 
        state: { message: 'Регистрация успешна! Войдите в систему' }
      });
    } catch (err) {
      console.error("ОШИБКА РЕГИСТРАЦИИ:", err);
      setError(
        err.response?.data?.error || 
        'Ошибка регистрации. Попробуйте другой логин или email'
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-container">
      <div className="auth-card fade-in">
        <div className="auth-header">
          <div className="logo">
            <span className="logo-icon">📦</span>
            <h1>DeliverTrack</h1>
          </div>
          <p className="auth-subtitle">Создайте аккаунт</p>
        </div>

        <form onSubmit={handleSubmit} className="auth-form">
          {error && (
            <div className="error-message">
              {error}
            </div>
          )}

          {/* Логин */}
          <div className="form-group">
            <label htmlFor="username">Логин</label>
            <input
              id="username"
              name="username"
              type="text"
              placeholder="Придумайте логин"
              value={formData.username}
              onChange={handleChange}
              required
              autoFocus
            />
          </div>

          {/* Email */}
          <div className="form-group">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              name="email"
              type="email"
              placeholder="your@email.com"
              value={formData.email}
              onChange={handleChange}
              required
            />
          </div>

          {/* Пароль */}
          <div className="form-group">
            <label htmlFor="password">Пароль</label>
            <input
              id="password"
              name="password"
              type="password"
              placeholder="Минимум 6 символов"
              value={formData.password}
              onChange={handleChange}
              required
            />
          </div>

          {/* Подтверждение пароля */}
          <div className="form-group">
            <label htmlFor="confirmPassword">Повторите пароль</label>
            <input
              id="confirmPassword"
              name="confirmPassword"
              type="password"
              placeholder="Введите пароль ещё раз"
              value={formData.confirmPassword}
              onChange={handleChange}
              required
            />
          </div>

          {/* Роль */}
          <div className="form-group">
            <label htmlFor="role">Я регистрируюсь как</label>
            <select
              id="role"
              name="role"
              value={formData.role}
              onChange={handleChange}
            >
              <option value="customer">Клиент (заказываю доставки)</option>
              <option value="courier">Курьер (выполняю доставки)</option>
            </select>
          </div>

          <button 
            type="submit" 
            className="btn btn-primary btn-lg"
            disabled={loading}
          >
            {loading ? (
              <>
                <span className="loading"></span>
                Регистрация...
              </>
            ) : (
              'Зарегистрироваться'
            )}
          </button>

          <div className="auth-footer">
            <p className="text-muted">
              Уже есть аккаунт?{' '}
              <Link to="/login" className="auth-link">
                Войти
              </Link>
            </p>
          </div>
        </form>
      </div>
    </div>
  );
}

export default Register;