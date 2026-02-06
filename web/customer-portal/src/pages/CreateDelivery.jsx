// Страница создания новой доставки
import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { createDelivery, getCurrentUser } from '../services/api';
import '../styles/create-delivery.css';

function CreateDelivery() {
  const navigate = useNavigate();
  const user = getCurrentUser();
  
  // Состояние формы
  const [formData, setFormData] = useState({
    pickup_address: '',
    delivery_address: '',
    pickup_lat: '',
    pickup_lng: '',
    delivery_lat: '',
    delivery_lng: '',
    scheduled_date: '',
    notes: ''
  });
  
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

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
    setLoading(true);

    try {
      // Формируем данные для отправки
      const deliveryData = {
        customer_id: user.customer_id || user.id,
        // Формат POINT для PostgreSQL: "POINT(longitude latitude)"
        pickup_location: `POINT(${formData.pickup_lng || -122.4194} ${formData.pickup_lat || 37.7749})`,
        delivery_location: `POINT(${formData.delivery_lng || -122.4089} ${formData.delivery_lat || 37.7849})`,
        scheduled_date: formData.scheduled_date || new Date().toISOString(),
        notes: formData.notes
      };

      // Отправляем на бэкенд
      const newDelivery = await createDelivery(deliveryData);
      
      // Перенаправляем на страницу отслеживания
      navigate(`/track/${newDelivery.id}`);
    } catch (err) {
      console.error(err);
      setError(
        err.response?.data?.error || 
        'Не удалось создать доставку. Проверьте данные'
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="create-delivery-page">
      {/* Шапка */}
      <header className="page-header">
        <div className="container">
          <Link to="/dashboard" className="back-link">
            ← Назад
          </Link>
          <h1>Новая доставка</h1>
        </div>
      </header>

      {/* Форма */}
      <main className="page-content">
        <div className="container">
          <div className="form-container card fade-in">
            {error && (
              <div className="error-message mb-3">
                {error}
              </div>
            )}

            <form onSubmit={handleSubmit}>
              {/* Секция: Откуда */}
              <div className="form-section">
                <h3 className="section-title">
                  <span className="section-icon">🏠</span>
                  Откуда забрать
                </h3>

                <div className="form-group">
                  <label htmlFor="pickup_address">Адрес забора</label>
                  <input
                    id="pickup_address"
                    name="pickup_address"
                    type="text"
                    placeholder="ул. Пушкина, д. 10"
                    value={formData.pickup_address}
                    onChange={handleChange}
                    required
                  />
                  <small className="text-muted">
                    Полный адрес откуда нужно забрать посылку
                  </small>
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label htmlFor="pickup_lat">Широта (необязательно)</label>
                    <input
                      id="pickup_lat"
                      name="pickup_lat"
                      type="number"
                      step="any"
                      placeholder="37.7749"
                      value={formData.pickup_lat}
                      onChange={handleChange}
                    />
                  </div>

                  <div className="form-group">
                    <label htmlFor="pickup_lng">Долгота (необязательно)</label>
                    <input
                      id="pickup_lng"
                      name="pickup_lng"
                      type="number"
                      step="any"
                      placeholder="-122.4194"
                      value={formData.pickup_lng}
                      onChange={handleChange}
                    />
                  </div>
                </div>
              </div>

              {/* Разделитель */}
              <div className="route-divider">
                <div className="divider-line"></div>
                <span className="divider-icon">🚚</span>
                <div className="divider-line"></div>
              </div>

              {/* Секция: Куда */}
              <div className="form-section">
                <h3 className="section-title">
                  <span className="section-icon">📍</span>
                  Куда доставить
                </h3>

                <div className="form-group">
                  <label htmlFor="delivery_address">Адрес доставки</label>
                  <input
                    id="delivery_address"
                    name="delivery_address"
                    type="text"
                    placeholder="пр. Ленина, д. 25, кв. 10"
                    value={formData.delivery_address}
                    onChange={handleChange}
                    required
                  />
                  <small className="text-muted">
                    Полный адрес куда нужно доставить
                  </small>
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label htmlFor="delivery_lat">Широта (необязательно)</label>
                    <input
                      id="delivery_lat"
                      name="delivery_lat"
                      type="number"
                      step="any"
                      placeholder="37.7849"
                      value={formData.delivery_lat}
                      onChange={handleChange}
                    />
                  </div>

                  <div className="form-group">
                    <label htmlFor="delivery_lng">Долгота (необязательно)</label>
                    <input
                      id="delivery_lng"
                      name="delivery_lng"
                      type="number"
                      step="any"
                      placeholder="-122.4089"
                      value={formData.delivery_lng}
                      onChange={handleChange}
                    />
                  </div>
                </div>
              </div>

              {/* Секция: Детали */}
              <div className="form-section">
                <h3 className="section-title">
                  <span className="section-icon">📝</span>
                  Дополнительная информация
                </h3>

                <div className="form-group">
                  <label htmlFor="scheduled_date">Желаемая дата и время</label>
                  <input
                    id="scheduled_date"
                    name="scheduled_date"
                    type="datetime-local"
                    value={formData.scheduled_date}
                    onChange={handleChange}
                  />
                  <small className="text-muted">
                    Оставьте пустым для доставки как можно скорее
                  </small>
                </div>

                <div className="form-group">
                  <label htmlFor="notes">Примечания для курьера</label>
                  <textarea
                    id="notes"
                    name="notes"
                    rows="4"
                    placeholder="Домофон не работает, звоните по телефону..."
                    value={formData.notes}
                    onChange={handleChange}
                  ></textarea>
                </div>
              </div>

              {/* Кнопки */}
              <div className="form-actions">
                <Link to="/dashboard" className="btn btn-secondary">
                  Отмена
                </Link>
                <button 
                  type="submit" 
                  className="btn btn-primary btn-lg"
                  disabled={loading}
                >
                  {loading ? (
                    <>
                      <span className="loading"></span>
                      Создание...
                    </>
                  ) : (
                    <>
                      <span>✓</span>
                      Создать доставку
                    </>
                  )}
                </button>
              </div>
            </form>
          </div>

          {/* Подсказка */}
          <div className="help-box mt-3">
            <p className="text-sm text-muted">
              💡 <strong>Подсказка:</strong> Координаты можно узнать на картах Google или Яндекс.
              Если не указать координаты, будут использованы координаты по умолчанию (Сан-Франциско).
            </p>
          </div>
        </div>
      </main>
    </div>
  );
}

export default CreateDelivery;