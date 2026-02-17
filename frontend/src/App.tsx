import { Events, Window } from "@wailsio/runtime";
import { useEffect, useRef, useState } from "react";
import { UserWindow } from "../bindings/tabswitcher";
import { clsx } from "./utils";

function App() {
  const [windowsState, setWindowsState] = useState<{
    windows: UserWindow[];
    selectedWindow: UserWindow["Hwnd"] | null;
  }>({
    windows: [],
    selectedWindow: null,
  });
  const iconsContainer = useRef<HTMLDivElement>(null);

  function activateWindow(hwnd: UserWindow["Hwnd"]) {
    setWindowsState((prevState) => ({
      ...prevState,
      selectedWindow: hwnd,
    }));
    Events.Emit("activateWindow", hwnd);
  }

  useEffect(() => {
    const body = document.body;

    const unregisterWindowsChanged = Events.On("userWindowsChanged", (event) => {
      const windows = (event.data ?? []) as UserWindow[];
      windows.sort((a, b) => b.LastActive - a.LastActive);

      setWindowsState((prevState) => {
        const { selectedWindow } = prevState;
        const stillExists =
          selectedWindow && windows.some((window: UserWindow) => window.Hwnd === selectedWindow);
        if (stillExists) {
          return {
            windows: windows,
            selectedWindow,
          };
        }

        const currentForeground = windows.find((window: UserWindow) => window.IsForeground);
        if (currentForeground) {
          return {
            windows: windows,
            selectedWindow: currentForeground.Hwnd,
          };
        }

        return {
          windows: windows,
          selectedWindow: windows[0]?.Hwnd || null,
        };
      });
    });

    const unregisterSystemKeyPressed = Events.On("systemKeyPressed", (event) => {
      if (event.data === "tab") {
        setWindowsState((prevState) => {
          const direction = "next";
          const { windows, selectedWindow } = prevState;
          const currentIndex = windows.findIndex((window) => window.Hwnd === selectedWindow);

          let nextIndex;
          if (direction === "next") {
            nextIndex = (currentIndex + 1) % (windows.length || 1);
          } else {
            nextIndex = (currentIndex - 1 + (windows.length || 1)) % (windows.length || 1);
          }

          return {
            ...prevState,
            selectedWindow: windows[nextIndex]?.Hwnd ?? windows[0]?.Hwnd,
          };
        });
      }
    });

    Window.OpenDevTools();

    const resizeInterval = setInterval(() => {
      Window.SetSize(
        iconsContainer.current?.clientWidth || 0,
        iconsContainer.current?.clientHeight || 0
      );

      Window.SetPosition(
        window.screen.availWidth / 2 - (body.clientWidth || 0) / 2,
        window.screen.availHeight / 2 - (body.clientHeight || 0) / 2
      );
    });

    return () => {
      clearInterval(resizeInterval);
      unregisterWindowsChanged();
      unregisterSystemKeyPressed();
    };
  }, []);

  return (
    <div ref={iconsContainer} className="absolute left-0 top-0 flex gap-1 p-3">
      {windowsState.windows.map((window) => (
        <div
          key={window.Hwnd}
          className={clsx(
            "cursor-pointer rounded-lg p-2 hover:bg-gray-200",
            window.Hwnd === windowsState.selectedWindow && "!bg-gray-100"
          )}
          onClick={() => activateWindow(window.Hwnd)}
        >
          <img src={window.IconBase64} alt="icon" className="size-10 max-w-max" />
        </div>
      ))}
    </div>
  );
}

export default App;
