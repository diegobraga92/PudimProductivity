import { useEffect, useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';
import { useStore } from './store/useStore';

export default function App() {
  const isAuthenticated = useStore(state => state.isAuthenticated);
  const fetchLists = useStore(state => state.fetchLists);
  const isServerDown = useStore(state => state.isServerDown);

  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const initializeApp = () => {
      if (isAuthenticated) {
        void fetchLists();
      }

      // Never block initial render on backend availability.
      setIsLoading(false);
    };

    initializeApp();
  }, [isAuthenticated, fetchLists]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-lg text-gray-600">Loading OpenToDo App...</div>
      </div>
    );
  }
  return (
    <Router>
      <Toaster position="top-right" />
      {isServerDown && (
        <div className="bg-red-600 text-white text-center text-sm font-medium px-4 py-2">
          Server is down. Showing local data until backend is available.
        </div>
      )}
      <Routes>
        <Route path="/login" element={!isAuthenticated ? <LoginPage /> : <Navigate to="/" />} />
        <Route path="/" element={isAuthenticated ? <DashboardPage /> : <Navigate to="/login" />} />
        <Route path="*" element={<Navigate to={isAuthenticated ? '/' : '/login'} />} />
      </Routes>
    </Router>
  );
}
// TODO
