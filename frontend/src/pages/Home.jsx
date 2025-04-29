import { useEffect, useState } from "react";
import "./Home.css";

/**
 * Interfaz principal del controlador: 
 *  1. Lista de AGFs registrados (solo informativa).  
 *  2. Lista de usuarios, con selector de AGF destino y botón “Iniciar Handover”.
 */
function Home() {
  const [agfList, setAgfList] = useState([]);
  const [userList, setUserList] = useState([]);
  const [message, setMessage] = useState("");

  /** ─────────────────────────  LOAD DATA  ───────────────────────── **/
  useEffect(() => {
    // AGFs
    fetch("http://138.4.21.21:8080/agfs")
      .then((res) => res.json())
      .then((data) => Array.isArray(data.agfs) && setAgfList(data.agfs))
      .catch((err) => console.error("Error fetching AGFs:", err));

    // Usuarios
    fetch("http://138.4.21.21:8080/users")
      .then((res) => res.json())
      .then((data) => Array.isArray(data.users) && setUserList(data.users))
      .catch((err) => console.error("Error fetching users:", err));
  }, []);

  /** ────────────────────  TRIGGER HANDOVER  ───────────────────── **/
  const handleTriggerHandover = (supi, gnbIdTarget) => {
    fetch("http://138.4.21.21:8080/triggerHandover", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ supi, gnbId: gnbIdTarget }),
    })
      .then((res) => res.json())
      .then((data) => setMessage(data.message))
      .catch((err) => {
        console.error("Error activando Handover:", err);
        setMessage("Error activando Handover");
      });
  };

  /** ───────────────────────────  RENDER  ───────────────────────── **/
  return (
    <div className="home-container">
      <h1>Controller</h1>

      {/* ---------- AGFs ---------- */}
      <p>Registered AGFs</p>      
      <table>
        <thead>
          <tr>
            <th>gnbId (hex)</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {agfList.length > 0 ? (
            agfList.map((agf) => (
              <tr key={agf.gnbId}>
                <td>{agf.gnbId}</td>
                <td>—</td>
              </tr>
            ))
          ) : (
            <tr className="no-data">
              <td colSpan="2">No AGFs registered.</td>
            </tr>
          )}
        </tbody>
      </table>

      {/* ---------- MENSAJE ---------- */}
      {message && <p>{message}</p>}

      {/* ---------- USUARIOS ---------- */}
      <p>Registered Users</p>
      <table>
        <thead>
          <tr>
            <th>IMSI</th>
            <th>SUPI</th>
            <th>Current AGF</th>
            <th>Target AGF</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {userList.length > 0 ? (
            userList.map((user) => (
              <UserRow
                key={user.supi}
                user={user}
                agfList={agfList}
                onHandover={handleTriggerHandover}
              />
            ))
          ) : (
            <tr className="no-data">
              <td colSpan="5">No users registered.</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

/* ─────────────────────────────  UserRow  ─────────────────────────── */
function UserRow({ user, agfList, onHandover }) {
  const [target, setTarget] = useState(agfList[0]?.gnbId || "");

  return (
    <tr>
      <td>{user.imsi}</td>
      <td>{user.supi}</td>
      <td>{user.gnb_id}</td>
      <td>
        <select value={target} onChange={(e) => setTarget(e.target.value)}>
          {agfList.map((agf) => (
            <option key={agf.gnbId} value={agf.gnbId}>
              {agf.gnbId}
            </option>
          ))}
        </select>
      </td>
      <td>
        <button
          className="btn-handover"
          onClick={() => onHandover(user.supi, target)}
        >
          Trigger Handover
        </button>
      </td>
    </tr>
  );
}

export default Home;