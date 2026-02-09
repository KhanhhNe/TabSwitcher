import { Events, WML } from "@wailsio/runtime";
import { useEffect, useState } from "react";
import { UserWindow } from "../bindings/changeme";

function App() {
  const [windows, setWindows] = useState<UserWindow[]>([]);

  useEffect(() => {
    Events.On("userWindowsChanged", (event) => {
      setWindows(event.data)
    })
    // Reload WML so it picks up the wml tags
    WML.Reload();
  }, []);

  return (
    <div className="flex gap-2">
      {windows.map((window) => (
        <div key={window.Hwnd}>
          <img src={`data:image/png;base64,${window.IconBase64}`} alt="icon" />
          {/* <div>{window.Caption}</div> */}
        </div>
      ))}
    </div>
  );
}

export default App;
