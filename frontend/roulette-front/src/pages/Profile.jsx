import { useLocation, useNavigate } from 'react-router-dom'

export default function Profile() {
  const location = useLocation()
  const navigate = useNavigate()
  const { userData } = location.state || {}

  return (
    <div className="profile-page">
      <button onClick={() => navigate(-1)} className="close-button">
        ← Назад
      </button>
      
      <div className="profile-header">
        <img src={userData?.avatar} alt="Profile" className="profile-avatar" />
        <h2>{userData?.firstName}</h2>
      </div>
    </div>
  )
}