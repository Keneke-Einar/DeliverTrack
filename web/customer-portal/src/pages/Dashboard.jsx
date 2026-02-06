// Главная страница - дашборд с доставками
import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { getDeliveries, getCurrentUser, logout } from '../services/api';
import '../styles/dashboard.css';

function Dashboard() {
  const navigate = useNavigate();
  const user = getCurrentUser(); // Получаем данные пользователя
  
  // Состояния
  const [deliveries, setDeliveries] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState('all'); // Фильтр по статусу

  // Загружаем доставки при монтировании компонента
  useEffect(() => {
    loadDeliveries();
  }, [filter]);

  // Функция загрузки доставок
  const loadDeliveries = async () => {
    try {
      setLoading(true);
      
      // Параметры фильтрации
      const params = {};
      if (filter !== 'all') {
        params.status = filter;
      }
      
      // Для клиентов - показываем только их доставки
      if (user.role === 'customer' && user.customer_id) {
        params.customer_id = user.customer_id;
      }
      
      // Загружаем с бэкенда
      const data = await getDeliveries(params);
      setDeliveries(data || []);
    } catch (err) {
      setError('Не удалось загрузить доставки');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  // Обработчик выхода
  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  // Перевод статусов на русский
  const statusText = {
    pending: 'Ожидает',
    assigned: 'Назначена',
    picked_up: 'Забрана',
    in_transit: 'В пути',
    delivered: 'Доставлена',
    cancelled: 'Отменена'
  };

  return (
    <div className="dashboard">
      {/* Шапка */}
      <header className="dashboard-header">
        <div className="container">
          <div className="header-content">
            <div className="logo">
              <span className="logo-icon">📦</span>
              <h2>DeliverTrack</h2>
            </div>
            
            <div className="header-actions">
              <span className="user-info">
                Привет, <strong>{user.username}</strong>
              </span>
              <button onClick={handleLogout} className="btn btn-secondary btn-sm">
                Выход
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Основной контент */}
      <main className="dashboard-main">
        <div className="container">
          {/* Заголовок и кнопка создания */}
          <div className="dashboard-top">
            <div>
              <h1>Мои доставки</h1>
              <p className="text-muted">
                Всего: {deliveries.length}
              </p>
            </div>
            
            <Link to="/create-delivery" className="btn btn-primary">
              <span>➕</span>
              Новая доставка
            </Link>
          </div>

          {/* Фильтры */}
          <div className="filters">
            <button 
              className={`filter-btn ${filter === 'all' ? 'active' : ''}`}
              onClick={() => setFilter('all')}
            >
              Все
            </button>
            <button 
              className={`filter-btn ${filter === 'pending' ? 'active' : ''}`}
              onClick={() => setFilter('pending')}
            >
              Ожидает
            </button>
            <button 
              className={`filter-btn ${filter === 'in_transit' ? 'active' : ''}`}
              onClick={() => setFilter('in_transit')}
            >
              В пути
            </button>
            <button 
              className={`filter-btn ${filter === 'delivered' ? 'active' : ''}`}
              onClick={() => setFilter('delivered')}
            >
              Доставлено
            </button>
          </div>

          {/* Список доставок */}
          <div className="deliveries-grid">
            {loading ? (
              <div className="loading-container">
                <span className="loading"></span>
                <p>Загрузка...</p>
              </div>
            ) : error ? (
              <div className="error-message">{error}</div>
            ) : deliveries.length === 0 ? (
              <div className="empty-state">
                <span className="empty-icon">📭</span>
                <h3>Доставок нет</h3>
                <p className="text-muted">
                  Создайте свою первую доставку!
                </p>
                <Link to="/create-delivery" className="btn btn-primary mt-2">
                  Создать доставку
                </Link>
              </div>
            ) : (
              deliveries.map((delivery) => (
                <div key={delivery.ID} className="delivery-card card fade-in">
                  {/* Номер доставки */}
                  <div className="delivery-header">
                    <span className="delivery-number mono">
                      #{delivery.ID}
                    </span>
                    <span className={`badge badge-${delivery.Status}`}>
                      {statusText[delivery.Status]}
                    </span>
                  </div>

                  {/* Информация */}
                  <div className="delivery-info">
                    <div className="delivery-route">
                      <div className="route-point">
                        <span className="route-icon">🏠</span>
                        <div>
                          <p className="text-sm text-muted">Откуда</p>
                          <p className="route-address">
                            {delivery.PickupLocation || 'Адрес забора'}
                          </p>
                        </div>
                      </div>
                      
                      <div className="route-line"></div>
                      
                      <div className="route-point">
                        <span className="route-icon">📍</span>
                        <div>
                          <p className="text-sm text-muted">Куда</p>
                          <p className="route-address">
                            {delivery.DeliveryLocation || 'Адрес доставки'}
                          </p>
                        </div>
                      </div>
                    </div>

                    {/* Примечания */}
                    {delivery.Notes && (
                      <p className="delivery-notes">
                        <strong>Примечание:</strong> {delivery.Notes}
                      </p>
                    )}

                    {/* Дата */}
                    <p className="text-sm text-muted">
                      Создана: {(() => {
                        try {
                          const date = new Date(delivery.CreatedAt);
                          return isNaN(date.getTime()) 
                            ? 'Неверная дата'
                            : date.toLocaleDateString('ru-RU', {
                                year: 'numeric',
                                month: 'long',
                                day: 'numeric',
                                hour: '2-digit',
                                minute: '2-digit'
                              });
                        } catch (e) {
                          return 'Неверная дата';
                        }
                      })()}
                    </p>
                  </div>

                  {/* Кнопка отслеживания */}
                  <Link 
                    to={`/track/${delivery.ID}`} 
                    className="btn btn-primary btn-sm"
                  >
                    <span>🗺️</span>
                    Отследить
                  </Link>
                </div>
              ))
            )}
          </div>
        </div>
      </main>
    </div>
  );
}

export default Dashboard;