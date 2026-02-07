import axios from 'axios';

// Use environment variable for API URL, fallback to empty for proxy in development
const API_BASE_URL = import.meta.env.VITE_API_URL || '';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Добавляем токен к запросам
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401 && !error.config.url.includes('/login')) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// === 🛠 ВСПОМОГАТЕЛЬНАЯ ФУНКЦИЯ ===
// Распаковываем данные из JWT токена (Base64 decode)
function parseJwt(token) {
  try {
    const base64Url = token.split('.')[1];
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function(c) {
        return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
    }).join(''));

    const data = JSON.parse(jsonPayload);
    
    // Приводим данные к удобному виду
    return {
      id: data.user_id || data.sub, // ID пользователя
      username: data.username || data.sub,
      role: data.role,
      customer_id: data.customer_id, 
      courier_id: data.courier_id
    };
  } catch (e) {
    console.error("Ошибка парсинга токена:", e);
    return null;
  }
}

// === АУТЕНТИФИКАЦИЯ ===

export const register = async (username, email, password, role = 'customer') => {
  const response = await api.post('/api/auth/register', {
    username,
    email,
    password,
    role
  });
  return response.data;
};

export const login = async (username, password) => {
  const response = await api.post('/api/auth/login', { username, password });
  
  if (response.data.token) {
    const token = response.data.token;
    localStorage.setItem('token', token);
    
    // РЕШЕНИЕ ПРОБЛЕМЫ 404:
    // Вместо запроса к серверу, берем данные прямо из токена!
    const userData = parseJwt(token);
    
    if (userData) {
      localStorage.setItem('user', JSON.stringify(userData));
      console.log("Успешный вход. Данные из токена:", userData);
    }
  }
  return response.data;
};

export const logout = () => {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
  window.location.href = '/login';
};

// Временно возвращаем фейкового пользователя всегда
export const getCurrentUser = () => {
  return {
    username: "Super Designer",
    email: "design@test.com",
    role: "customer", // Можешь поменять на 'courier', чтобы увидеть интерфейс курьера
    id: 1,
    customer_id: 1
  };
};
// === ДОСТАВКИ ===

export const getDeliveries = async (params = {}) => (await api.get('/deliveries', { params })).data;
export const getDeliveryById = async (id) => (await api.get(`/deliveries/${id}`)).data;

// РЕШЕНИЕ ПРОБЛЕМЫ 405:
// Убрали лишний слеш в конце адреса ('/api/delivery/deliveries' вместо '.../')
export const createDelivery = async (data) => (await api.post('/deliveries', data)).data;

export const updateDeliveryStatus = async (id, status, notes = '') => (await api.put(`/deliveries/${id}/status`, { status, notes })).data;

// === ОТСЛЕЖИВАНИЕ ===
export const getDeliveryTrack = async (deliveryId) => (await api.get(`/tracking/deliveries/${deliveryId}/track`)).data;
export const getDeliveryLocation = async (deliveryId) => (await api.get(`/tracking/deliveries/${deliveryId}/location`)).data;
export const calculateETA = async (deliveryId, currentLocation) => (await api.post(`/tracking/deliveries/${deliveryId}/eta`, { current_location: currentLocation })).data;
export const getNotifications = async () => (await api.get('/notification/notifications')).data;
export const markNotificationAsRead = async (notificationId) => (await api.put(`/notification/notifications/${notificationId}/read`)).data;

export default api;