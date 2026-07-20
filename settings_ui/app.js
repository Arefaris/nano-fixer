document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('settings-form');
    const apiKeyInput = document.getElementById('api-key');
    const apiBaseInput = document.getElementById('api-base');
    const modelNameInput = document.getElementById('model-name');
    const modCtrl = document.getElementById('mod-ctrl');
    const modAlt = document.getElementById('mod-alt');
    const modShift = document.getElementById('mod-shift');
    const modWin = document.getElementById('mod-win');
    const hotkeyKey = document.getElementById('hotkey-key');
    const targetLang = document.getElementById('target-lang');
    const autostart = document.getElementById('autostart');
    
    const toggleKeyBtn = document.getElementById('toggle-key');
    const btnSave = document.getElementById('btn-save');
    const btnLogs = document.getElementById('btn-logs');
    const toast = document.getElementById('toast');

    // Get token from URL
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token') || '';

    // Show/hide API Key
    toggleKeyBtn.addEventListener('click', () => {
        const isPassword = apiKeyInput.type === 'password';
        apiKeyInput.type = isPassword ? 'text' : 'password';
        
        const svg = toggleKeyBtn.querySelector('svg');
        if (isPassword) {
            // eye-off icon
            svg.innerHTML = '<path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path><line x1="1" y1="1" x2="23" y2="23"></line>';
        } else {
            // eye icon
            svg.innerHTML = '<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle>';
        }
    });

    // Show/hide eye button based on user typing a new key
    apiKeyInput.addEventListener('input', () => {
        if (apiKeyInput.value.length > 0) {
            toggleKeyBtn.style.display = 'block';
        } else if (apiKeyInput.placeholder.includes('API Key is saved')) {
            toggleKeyBtn.style.display = 'none';
        }
    });

    // Helper to fetch config
    async function loadConfig() {
        try {
            const res = await fetch(`/api/config?token=${token}`);
            if (!res.ok) throw new Error('Failed to load settings');
            
            const config = await res.json();
            
            if (config.APIKey === '••••••••••••') {
                apiKeyInput.value = '';
                apiKeyInput.placeholder = '•••••••••••• (API Key is saved)';
                toggleKeyBtn.style.display = 'none';
            } else {
                apiKeyInput.value = config.APIKey || '';
                apiKeyInput.placeholder = 'sk-proj-...';
                toggleKeyBtn.style.display = 'block';
            }
            apiBaseInput.value = config.APIBaseURL || '';
            modelNameInput.value = config.ModelName || 'gpt-4o-mini';
            
            // Parse hotkey modifiers e.g. "shift+alt"
            const mods = (config.HotkeyMod || '').toLowerCase().split('+');
            modCtrl.checked = mods.includes('ctrl');
            modAlt.checked = mods.includes('alt');
            modShift.checked = mods.includes('shift');
            modWin.checked = mods.includes('win');
            
            hotkeyKey.value = config.HotkeyKey || 'C';
            targetLang.value = config.TargetLanguage || 'Auto';
            autostart.checked = config.Autostart || false;
            
        } catch (err) {
            showToast('Error loading settings: ' + err.message, true);
        }
    }

    // Save configuration
    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        btnSave.classList.add('btn-loading');
        btnSave.disabled = true;

        // Reconstruct modifiers string
        const mods = [];
        if (modCtrl.checked) mods.push('ctrl');
        if (modAlt.checked) mods.push('alt');
        if (modShift.checked) mods.push('shift');
        if (modWin.checked) mods.push('win');
        const hotkeyModStr = mods.join('+');

        let sendKey = apiKeyInput.value;
        if (sendKey === '') {
            if (apiKeyInput.placeholder.includes('API Key is saved')) {
                sendKey = '••••••••••••';
            } else {
                showToast('API Key is required!', true);
                btnSave.classList.remove('btn-loading');
                btnSave.disabled = false;
                return;
            }
        }

        const payload = {
            APIKey: sendKey,
            APIBaseURL: apiBaseInput.value,
            ModelName: modelNameInput.value,
            HotkeyMod: hotkeyModStr,
            HotkeyKey: hotkeyKey.value,
            TargetLanguage: targetLang.value,
            Autostart: autostart.checked
        };

        try {
            const res = await fetch(`/api/config?token=${token}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!res.ok) throw new Error('Failed to save settings');
            
            showToast('Settings saved successfully!');
            
            // Optional: Close page after a small delay
            setTimeout(() => {
                window.close();
            }, 1000);
            
        } catch (err) {
            showToast('Error saving settings: ' + err.message, true);
        } finally {
            btnSave.classList.remove('btn-loading');
            btnSave.disabled = false;
        }
    });

    // View Logs button click
    btnLogs.addEventListener('click', async () => {
        try {
            const res = await fetch(`/api/logs?token=${token}`, { method: 'POST' });
            if (!res.ok) throw new Error('Failed to open logs');
        } catch (err) {
            showToast('Error opening logs: ' + err.message, true);
        }
    });

    function showToast(message, isError = false) {
        toast.textContent = message;
        if (isError) {
            toast.style.borderColor = '#ff4a4a';
            toast.style.boxShadow = '0 0 20px rgba(255, 74, 74, 0.2)';
        } else {
            toast.style.borderColor = '#00ffaa';
            toast.style.boxShadow = '0 0 20px rgba(0, 255, 170, 0.2)';
        }
        toast.classList.add('show');
        setTimeout(() => {
            toast.classList.remove('show');
        }, 3000);
    }

    // Load initially
    loadConfig();
});
