function applyThemeToBody() {
    const body = document.body;
    const toggleBtn = document.getElementById("themeToggle");
    const savedTheme = localStorage.getItem("theme") || "light";
    
    body.setAttribute("data-theme", savedTheme);
    if (toggleBtn) toggleBtn.textContent = savedTheme === "dark" ? "☀️" : "🌙";
}

function initThemeToggle() {
    const body = document.body;
    const toggleBtn = document.getElementById("themeToggle");
    applyThemeToBody();

    if (!toggleBtn) return;

    toggleBtn.addEventListener("click", () => {
        const currentTheme = body.getAttribute("data-theme");
        const newTheme = currentTheme === "light" ? "dark" : "light";
        body.setAttribute("data-theme", newTheme);
        localStorage.setItem("theme", newTheme);
        toggleBtn.textContent = newTheme === "dark" ? "☀️" : "🌙";

        if (window.graphVisualizer && window.graphVisualizer.network) {
            window.graphVisualizer.applyThemeToNetwork();
        }
    });
}
