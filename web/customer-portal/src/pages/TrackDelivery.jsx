// Страница отслеживания доставки с картой
import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { MapContainer, TileLayer, Marker, Popup, Polyline } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import { getDeliveryById, getDeliveryTrack } from '../services/api';
import '../styles/track-delivery.css';

// Иконки для маркеров (fix для Leaflet в React)
import L from 'leaflet';
import icon from 'leaflet/dist/images/marker-icon.png';
import iconShadow from 'leaflet/dist/images/marker-shadow.png';

let DefaultIcon = L.icon({
  iconUrl: icon,
  shadowUrl: iconShadow,
  iconSize: [25, 41],
  iconAnchor: [12, 41]
});

L.Marker.prototype.options.icon = DefaultIcon;

function TrackDelivery() {
  const { id } = useParams(); // Получаем ID из URL
  
  // Состояния
  const [delivery, setDelivery] = useState(null);
  const [track, setTrack] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Загружаем данные при монтировании
  useEffect(() => {
    loadDeliveryData();
  }, [id]);

  // Функция загрузки данных
  const loadDeliveryData = async () => {
    try {
      setLoading(true);
      
      // Загружаем данные доставки
      const deliveryData = await getDeliveryById(id);
      setDelivery(deliveryData);
      
      // Загружаем историю перемещений
      try {
        const trackData = await getDeliveryTrack(id);
        setTrack(trackData || []);
      } catch (err) {
        // Если нет данных трека - не проблема
        console.log('Нет данных трека:', err);
      }
    } catch (err) {
      setError('Не удалось загрузить данные доставки');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  // Парсим POINT формат из PostgreSQL: "POINT(-122.4194 37.7749)"
  const parsePoint = (pointStr) => {
    if (!pointStr) return null;
    const match = pointStr.match(/POINT\(([^ ]+) ([^ ]+)\)/);
    if (!match) return null;
    return {
      lng: parseFloat(match[1]),
      lat: parseFloat(match[2])
    };
  };

  // Статусы на русском
  const statusText = {
    pending: 'Ожидает',
    assigned: 'Назначена',
    picked_up: 'Забрана',
    in_transit: 'В пути',
    delivered: 'Доставлена',
    cancelled: 'Отменена'
  };

  if (loading) {
    return (
      <div className="loading-page">
        <span className="loading"></span>
        <p>Загрузка...</p>
      </div>
    );
  }

  if (error || !delivery) {
    return (
      <div className="error-page">
        <div className="error-content">
          <span className="error-icon">❌</span>
          <h2>{error || 'Доставка не найдена'}</h2>
          <Link to="/dashboard" className="btn btn-primary mt-2">
            Вернуться к доставкам
          </Link>
        </div>
      </div>
    );
  }

  // Парсим координаты
  const pickupPoint = parsePoint(delivery.pickup_location);
  const deliveryPoint = parsePoint(delivery.delivery_location);
  
  // Центр карты (между точками забора и доставки)
  const centerLat = pickupPoint && deliveryPoint 
    ? (pickupPoint.lat + deliveryPoint.lat) / 2 
    : 37.7749;
  const centerLng = pickupPoint && deliveryPoint 
    ? (pickupPoint.lng + deliveryPoint.lng) / 2 
    : -122.4194;

  // Путь курьера (если есть история)
  const pathCoordinates = track
    .map(point => {
      const coords = parsePoint(point.location);
      return coords ? [coords.lat, coords.lng] : null;
    })
    .filter(Boolean);

  return (
    <div className="track-delivery-page">
      {/* Шапка */}
      <header className="page-header">
        <div className="container">
          <Link to="/dashboard" className="back-link">
            ← Назад к доставкам
          </Link>
          <div className="header-title">
            <h1>Доставка #{delivery.id}</h1>
            <span className={`badge badge-${delivery.status}`}>
              {statusText[delivery.status]}
            </span>
          </div>
        </div>
      </header>

      {/* Основной контент */}
      <main className="track-content">
        <div className="track-layout">
          {/* Информационная панель слева */}
          <aside className="track-sidebar">
            <div className="info-card card">
              <h3>Информация о доставке</h3>
              
              {/* Маршрут */}
              <div className="route-info">
                <div className="route-point">
                  <span className="route-icon">🏠</span>
                  <div>
                    <p className="text-sm text-muted">Откуда</p>
                    <p className="route-address">
                      {delivery.pickup_location}
                    </p>
                  </div>
                </div>
                
                <div className="route-line"></div>
                
                <div className="route-point">
                  <span className="route-icon">📍</span>
                  <div>
                    <p className="text-sm text-muted">Куда</p>
                    <p className="route-address">
                      {delivery.delivery_location}
                    </p>
                  </div>
                </div>
              </div>

              {/* Примечания */}
              {delivery.notes && (
                <div className="notes-box">
                  <p className="text-sm text-muted">Примечания:</p>
                  <p>{delivery.notes}</p>
                </div>
              )}

              {/* Даты */}
              <div className="dates-info">
                <div className="date-item">
                  <span className="text-sm text-muted">Создана:</span>
                  <span className="text-sm">
                    {new Date(delivery.created_at).toLocaleString('ru-RU')}
                  </span>
                </div>
                
                {delivery.updated_at !== delivery.created_at && (
                  <div className="date-item">
                    <span className="text-sm text-muted">Обновлена:</span>
                    <span className="text-sm">
                      {new Date(delivery.updated_at).toLocaleString('ru-RU')}
                    </span>
                  </div>
                )}
              </div>

              {/* История перемещений */}
              {track.length > 0 && (
                <div className="track-history">
                  <h4>История перемещений</h4>
                  <div className="track-list">
                    {track.slice(0, 5).map((point, index) => (
                      <div key={index} className="track-item">
                        <span className="track-time">
                          {new Date(point.timestamp).toLocaleTimeString('ru-RU')}
                        </span>
                        <span className="track-coords text-sm text-muted mono">
                          {point.location}
                        </span>
                      </div>
                    ))}
                    {track.length > 5 && (
                      <p className="text-sm text-muted text-center mt-1">
                        и ещё {track.length - 5} точек...
                      </p>
                    )}
                  </div>
                </div>
              )}
            </div>
          </aside>

          {/* Карта справа */}
          <div className="track-map-container">
            <div className="map-wrapper">
              {pickupPoint && deliveryPoint ? (
                <MapContainer
                  center={[centerLat, centerLng]}
                  zoom={13}
                  style={{ height: '100%', width: '100%' }}
                >
                  {/* Слой карты от OpenStreetMap */}
                  <TileLayer
                    attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
                    url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                  />
                  
                  {/* Маркер точки забора */}
                  <Marker position={[pickupPoint.lat, pickupPoint.lng]}>
                    <Popup>
                      <strong>Точка забора</strong>
                      <br />
                      {delivery.pickup_location}
                    </Popup>
                  </Marker>
                  
                  {/* Маркер точки доставки */}
                  <Marker position={[deliveryPoint.lat, deliveryPoint.lng]}>
                    <Popup>
                      <strong>Точка доставки</strong>
                      <br />
                      {delivery.delivery_location}
                    </Popup>
                  </Marker>
                  
                  {/* Путь курьера (если есть) */}
                  {pathCoordinates.length > 0 && (
                    <Polyline 
                      positions={pathCoordinates} 
                      color="#2563eb"
                      weight={3}
                    />
                  )}
                </MapContainer>
              ) : (
                <div className="map-placeholder">
                  <p className="text-muted">
                    Не удалось отобразить карту.
                    <br />
                    Проверьте формат координат.
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

export default TrackDelivery;