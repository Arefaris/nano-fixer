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
    const useLocalAi = document.getElementById('use-local-ai');
    const dlContainer = document.getElementById('download-progress-container');
    const dlStatus = document.getElementById('download-status');
    const dlPctText = document.getElementById('download-pct-text');
    const dlBar = document.getElementById('download-bar');
    
    const toggleKeyBtn = document.getElementById('toggle-key');
    const btnSave = document.getElementById('btn-save');
    const btnLogs = document.getElementById('btn-logs');
    const toast = document.getElementById('toast');

    let downloadInterval = null;

    useLocalAi.addEventListener('change', async (e) => {
        if (e.target.checked) {
            let exists = false;
            if (window.checkLocalAIFiles) {
                exists = await window.checkLocalAIFiles();
            }
            if (!exists) {
                dlContainer.style.display = 'block';
                if (window.startLocalAIDownload) window.startLocalAIDownload();
                
                downloadInterval = setInterval(async () => {
                    if (window.getLocalAIDownloadProgress) {
                        const progressJson = await window.getLocalAIDownloadProgress();
                        const p = JSON.parse(progressJson);
                        dlStatus.textContent = p.status;
                        dlPctText.textContent = p.pct + '%';
                        dlBar.style.width = p.pct + '%';
                        
                        if (!p.downloading && p.pct === 100) {
                            clearInterval(downloadInterval);
                            setTimeout(() => { dlContainer.style.display = 'none'; }, 2000);
                        } else if (!p.downloading && p.pct === 0 && p.status.includes('Error')) {
                            clearInterval(downloadInterval);
                            useLocalAi.checked = false;
                            showToast(p.status, true);
                        }
                    }
                }, 500);
            }
        } else {
            dlContainer.style.display = 'none';
            if (downloadInterval) clearInterval(downloadInterval);
        }
    });

    // Show/hide API Key
    toggleKeyBtn.addEventListener('click', () => {
        const isPassword = apiKeyInput.type === 'password';
        apiKeyInput.type = isPassword ? 'text' : 'password';
        
        const svg = toggleKeyBtn.querySelector('svg');
        if (isPassword) {
            svg.innerHTML = '<path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path><line x1="1" y1="1" x2="23" y2="23"></line>';
        } else {
            svg.innerHTML = '<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle>';
        }
    });

    apiKeyInput.addEventListener('input', () => {
        if (apiKeyInput.value.length > 0) {
            toggleKeyBtn.style.display = 'block';
        } else if (apiKeyInput.placeholder.includes('API Key is saved')) {
            toggleKeyBtn.style.display = 'none';
        }
    });

    async function loadConfig() {
        try {
            let configStr = '';
            if (window.getConfig) {
                configStr = await window.getConfig();
            } else {
                const res = await fetch('/api/config');
                configStr = await res.text();
            }
            const config = JSON.parse(configStr);
            
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
            
            const mods = (config.HotkeyMod || '').toLowerCase().split('+');
            modCtrl.checked = mods.includes('ctrl');
            modAlt.checked = mods.includes('alt');
            modShift.checked = mods.includes('shift');
            modWin.checked = mods.includes('win');
            
            hotkeyKey.value = config.HotkeyKey || 'C';
            targetLang.value = config.TargetLanguage || 'Auto';
            autostart.checked = config.Autostart || false;
            useLocalAi.checked = config.UseLocalAI || false;
            
        } catch (err) {
            showToast('Error loading settings: ' + err.message, true);
        }
    }

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        btnSave.classList.add('btn-loading');
        btnSave.disabled = true;

        const mods = [];
        if (modCtrl.checked) mods.push('ctrl');
        if (modAlt.checked) mods.push('alt');
        if (modShift.checked) mods.push('shift');
        if (modWin.checked) mods.push('win');
        const hotkeyModStr = mods.join('+');

        let sendKey = apiKeyInput.value;
        if (sendKey === '' && apiKeyInput.placeholder.includes('API Key is saved')) {
            sendKey = '••••••••••••';
        }

        const payload = {
            APIKey: sendKey,
            APIBaseURL: apiBaseInput.value,
            ModelName: modelNameInput.value,
            HotkeyMod: hotkeyModStr,
            HotkeyKey: hotkeyKey.value,
            TargetLanguage: targetLang.value,
            Autostart: autostart.checked,
            UseLocalAI: useLocalAi.checked
        };

        try {
            if (window.saveConfig) {
                const ok = await window.saveConfig(JSON.stringify(payload));
                if (!ok) throw new Error('Failed to save settings');
            } else {
                const res = await fetch('/api/config', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                if (!res.ok) throw new Error('Failed to save settings');
            }
            
            showToast('Settings saved successfully!');
            setTimeout(() => {
                if (window.closeWindow) {
                    window.closeWindow();
                } else {
                    window.close();
                }
            }, 800);
            
        } catch (err) {
            showToast('Error saving settings: ' + err.message, true);
        } finally {
            btnSave.classList.remove('btn-loading');
            btnSave.disabled = false;
        }
    });

    btnLogs.addEventListener('click', async () => {
        try {
            if (window.openLogs) {
                window.openLogs();
            } else {
                await fetch('/api/logs', { method: 'POST' });
            }
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

    loadConfig();
});
