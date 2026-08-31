// ============================================
// Warba Bank — Proactive Opportunity Engine
// Frontend SPA Application
// ============================================

(function () {
    'use strict';

    // --- API Client ---
    const API = {
        base: '/api',

        async get(path) {
            const res = await fetch(this.base + path);
            const data = await res.json();
            if (!data.success) throw new Error(data.error || 'API Error');
            return data.data;
        },

        async post(path, body) {
            const res = await fetch(this.base + path, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: body ? JSON.stringify(body) : undefined,
            });
            const data = await res.json();
            if (!data.success) throw new Error(data.error || 'API Error');
            return data.data;
        },

        async patch(path, body) {
            const res = await fetch(this.base + path, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
            });
            const data = await res.json();
            if (!data.success) throw new Error(data.error || 'API Error');
            return data.data;
        },

        // Auth
        authConfig: () => API.get('/auth/config'),
        authMe: () => API.get('/auth/me'),
        authGoogle: (credential) => API.post('/auth/google', { credential }),
        authLogout: () => API.post('/auth/logout'),

        // Domain
        clients: () => API.get('/clients'),
        client: (id) => API.get('/clients/' + id),
        analyzeClient: (id) => API.post('/clients/' + id + '/analyze'),
        opportunities: (params) => {
            const q = new URLSearchParams(params).toString();
            return API.get('/opportunities' + (q ? '?' + q : ''));
        },
        updateOppStatus: (id, status) => API.patch('/opportunities/' + id, { status }),
        portfolioScan: () => API.post('/portfolio/scan'),
        portfolioSummary: () => API.get('/portfolio/summary'),
        products: () => API.get('/products'),
    };

    // --- State ---
    let state = {
        currentPage: 'dashboard',
        currentUser: null,
        authConfig: null,
        clients: [],
        opportunities: [],
        products: [],
        summary: null,
    };

    // --- Router ---
    function initRouter() {
        window.addEventListener('hashchange', handleRoute);
        handleRoute();
    }

    function handleRoute() {
        const hash = window.location.hash.slice(1) || 'dashboard';
        const parts = hash.split('/');
        const page = parts[0];
        const param = parts[1];

        state.currentPage = page;

        // Update nav
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.toggle('active', item.dataset.page === page || (page === 'client' && item.dataset.page === 'clients'));
        });

        // Update title
        const titles = {
            dashboard: 'Dashboard',
            clients: 'Client Explorer',
            client: 'Client Detail',
            opportunities: 'Opportunities',
            products: 'Product Catalog',
        };
        document.getElementById('page-title').textContent = titles[page] || 'Dashboard';

        // Render page
        switch (page) {
            case 'dashboard': renderDashboard(); break;
            case 'clients': renderClients(); break;
            case 'client': renderClientDetail(param); break;
            case 'opportunities': renderOpportunities(); break;
            case 'products': renderProducts(); break;
            default: renderDashboard();
        }
    }

    // --- Helpers ---
    function $(id) { return document.getElementById(id); }
    function container() { return $('page-container'); }

    function formatKWD(amount) {
        if (amount >= 1000000) return 'KWD ' + (amount / 1000000).toFixed(1) + 'M';
        if (amount >= 1000) return 'KWD ' + (amount / 1000).toFixed(0) + 'K';
        return 'KWD ' + amount.toFixed(0);
    }

    function formatNumber(n) {
        return new Intl.NumberFormat().format(n);
    }

    function timeAgo(dateStr) {
        const d = new Date(dateStr);
        const now = new Date();
        const days = Math.floor((now - d) / 86400000);
        if (days === 0) return 'Today';
        if (days === 1) return 'Yesterday';
        if (days < 7) return days + ' days ago';
        if (days < 30) return Math.floor(days / 7) + ' weeks ago';
        return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
    }

    function industryClass(industry) {
        const map = {
            'Oil & Gas': 'oilgas', 'Real Estate': 'realestate', 'Trading': 'trading',
            'Construction': 'construction', 'Technology': 'technology', 'Healthcare': 'healthcare',
            'Logistics': 'logistics', 'Food & Beverage': 'foodbev', 'Manufacturing': 'manufacturing',
            'Automotive': 'automotive', 'Financial Services': 'financial', 'Engineering': 'engineering',
            'Hospitality': 'hospitality', 'Petrochemicals': 'petrochemicals', 'Retail': 'retail',
            'Agriculture': 'agriculture', 'Media': 'media',
        };
        return map[industry] || 'default';
    }

    function industryColor(industry) {
        const map = {
            'Oil & Gas': '#f97316', 'Real Estate': '#a855f7', 'Trading': '#22d3ee',
            'Construction': '#fbbf24', 'Technology': '#60a5fa', 'Healthcare': '#34d399',
            'Logistics': '#f472b6', 'Food & Beverage': '#a3e635', 'Manufacturing': '#fb923c',
            'Automotive': '#818cf8', 'Financial Services': '#d4a853', 'Engineering': '#2dd4bf',
            'Hospitality': '#e879f9', 'Petrochemicals': '#fda4af', 'Retail': '#93c5fd',
            'Agriculture': '#4ade80', 'Media': '#c4b5fd',
        };
        return map[industry] || '#94a3b8';
    }

    function initials(name) {
        if (!name) return 'RM';
        return name.split(' ').slice(0, 2).map(w => w[0]).join('').toUpperCase();
    }

    function confidenceMeter(confidence, size = 52) {
        const pct = Math.round(confidence * 100);
        const r = (size - 8) / 2;
        const c = 2 * Math.PI * r;
        const offset = c * (1 - confidence);
        const color = confidence >= 0.8 ? '#34d399' : confidence >= 0.6 ? '#fbbf24' : '#ef4444';
        return `<div class="confidence-meter" style="width:${size}px;height:${size}px">
            <svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">
                <circle class="confidence-bg" cx="${size / 2}" cy="${size / 2}" r="${r}" stroke-dasharray="${c}" stroke-dashoffset="0"/>
                <circle class="confidence-fill" cx="${size / 2}" cy="${size / 2}" r="${r}" stroke="${color}" stroke-dasharray="${c}" stroke-dashoffset="${offset}"/>
            </svg>
            <span class="confidence-label">${pct}%</span>
        </div>`;
    }

    function urgencyBadge(urgency) {
        return `<span class="badge badge-${(urgency || 'medium').toLowerCase()}">${urgency}</span>`;
    }

    function statusBadge(status) {
        return `<span class="badge badge-${(status || 'new').toLowerCase()}">${status}</span>`;
    }

    function showToast(msg, type = 'info') {
        const tc = $('toast-container');
        const t = document.createElement('div');
        t.className = 'toast ' + type;
        t.textContent = msg;
        tc.appendChild(t);
        setTimeout(() => { t.style.opacity = '0'; t.style.transform = 'translateX(50px)'; setTimeout(() => t.remove(), 300); }, 4000);
    }

    function showLoading(el, msg = 'Loading...') {
        el.innerHTML = `<div class="loading-screen"><div class="spinner"></div><p>${msg}</p></div>`;
    }

    function showEmpty(el, msg, sub) {
        el.innerHTML = `<div class="empty-state">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M8 15h8M9 9h.01M15 9h.01"/></svg>
            <p>${msg}</p>
            ${sub ? `<p class="sub">${sub}</p>` : ''}
        </div>`;
    }

    // --- Authentication & User Profile ---

    async function initAuth() {
        try {
            const [cfg, me] = await Promise.all([
                API.authConfig().catch(() => ({ google_client_id: '', enabled: false })),
                API.authMe().catch(() => ({ authenticated: false, user: null })),
            ]);

            state.authConfig = cfg;
            state.currentUser = me.authenticated ? me.user : null;

            updateUserUI();

            // Set up Google Identity Services
            if (cfg.google_client_id && window.google && window.google.accounts && window.google.accounts.id) {
                google.accounts.id.initialize({
                    client_id: cfg.google_client_id,
                    callback: handleGoogleCredentialResponse,
                    auto_select: false,
                });
                renderGoogleButtons();
            } else if (cfg.google_client_id) {
                window.addEventListener('load', () => {
                    if (window.google && window.google.accounts && window.google.accounts.id) {
                        google.accounts.id.initialize({
                            client_id: cfg.google_client_id,
                            callback: handleGoogleCredentialResponse,
                            auto_select: false,
                        });
                        renderGoogleButtons();
                    }
                });
            }
        } catch (err) {
            console.warn('[Auth] init error:', err);
        }

        // Attach Logout Buttons
        $('btn-sidebar-logout')?.addEventListener('click', handleLogout);
        $('btn-header-logout')?.addEventListener('click', handleLogout);
        $('btn-login-modal')?.addEventListener('click', openLoginModal);
    }

    function renderGoogleButtons() {
        if (!state.currentUser && state.authConfig?.google_client_id) {
            const headerSlot = $('header-gsi-btn');
            if (headerSlot) {
                headerSlot.innerHTML = '';
                google.accounts.id.renderButton(headerSlot, {
                    theme: 'filled_black',
                    size: 'medium',
                    shape: 'pill',
                    text: 'signin_with',
                });
            }
        }
    }

    window.handleGoogleCredentialResponse = async function (response) {
        if (!response || !response.credential) return;
        try {
            showToast('Authenticating with Google...', 'info');
            const data = await API.authGoogle(response.credential);
            state.currentUser = data.user;
            updateUserUI();
            showToast(`Welcome, ${data.user.name}!`, 'success');
            $('modal-overlay').style.display = 'none';
        } catch (err) {
            showToast('Sign-in failed: ' + err.message, 'error');
        }
    };

    async function handleLogout() {
        try {
            await API.authLogout();
            state.currentUser = null;
            updateUserUI();
            showToast('Signed out successfully', 'info');
            if (state.authConfig?.google_client_id && window.google?.accounts?.id) {
                renderGoogleButtons();
            }
        } catch (err) {
            showToast('Logout failed: ' + err.message, 'error');
        }
    }

    function updateUserUI() {
        const u = state.currentUser;

        const userName = $('user-name');
        const userRole = $('user-role');
        const userAvatar = $('user-avatar');
        const btnSidebarLogout = $('btn-sidebar-logout');

        const headerGsiSlot = $('header-gsi-btn');
        const btnLoginModal = $('btn-login-modal');
        const headerUserPill = $('header-user-pill');
        const headerUserName = $('header-user-name');
        const headerUserAvatar = $('header-user-avatar');

        if (u) {
            if (userName) userName.textContent = u.name;
            if (userRole) userRole.textContent = u.role || 'Senior RM';
            if (userAvatar) {
                if (u.avatar) {
                    userAvatar.innerHTML = `<img src="${u.avatar}" alt="${u.name}" onerror="this.parentElement.textContent='${initials(u.name)}'" />`;
                } else {
                    userAvatar.textContent = initials(u.name);
                }
            }
            if (btnSidebarLogout) btnSidebarLogout.style.display = 'flex';

            if (headerGsiSlot) headerGsiSlot.style.display = 'none';
            if (btnLoginModal) btnLoginModal.style.display = 'none';
            if (headerUserPill) headerUserPill.style.display = 'inline-flex';
            if (headerUserName) headerUserName.textContent = u.name;
            if (headerUserAvatar) {
                if (u.avatar) {
                    headerUserAvatar.src = u.avatar;
                    headerUserAvatar.style.display = 'inline-block';
                } else {
                    headerUserAvatar.style.display = 'none';
                }
            }
        } else {
            if (userName) userName.textContent = 'Ahmad Al-Mutairi';
            if (userRole) userRole.textContent = 'Demo RM Profile';
            if (userAvatar) userAvatar.textContent = 'AM';
            if (btnSidebarLogout) btnSidebarLogout.style.display = 'none';

            if (headerUserPill) headerUserPill.style.display = 'none';

            if (state.authConfig?.google_client_id) {
                if (headerGsiSlot) headerGsiSlot.style.display = 'flex';
                if (btnLoginModal) btnLoginModal.style.display = 'none';
                renderGoogleButtons();
            } else {
                if (headerGsiSlot) headerGsiSlot.style.display = 'none';
                if (btnLoginModal) btnLoginModal.style.display = 'inline-flex';
            }
        }
    }

    function openLoginModal() {
        const modal = $('modal');
        const modalTitle = $('modal-title');
        const modalBody = $('modal-body');
        const modalOverlay = $('modal-overlay');

        modalTitle.textContent = 'Google Authentication';
        modalBody.innerHTML = `
            <div class="auth-modal-card">
                <div class="auth-modal-icon">W</div>
                <h3 class="auth-modal-title">Sign in to Warba Opportunity Engine</h3>
                <p class="auth-modal-desc">
                    Authenticate using your official Google account to access corporate relationship portfolios and AI recommendations.
                </p>
                <div class="auth-btn-group">
                    <div id="modal-gsi-slot" class="gsi-button-wrapper"></div>
                    <button class="btn btn-primary" id="btn-demo-signin" style="width:100%">
                        ⚡ Quick Demo Sign-In (Ahmad Al-Mutairi)
                    </button>
                    ${!state.authConfig?.google_client_id ? `
                        <div style="font-size:0.75rem;color:var(--text-muted);margin-top:8px;line-height:1.4">
                            💡 <em>To enable direct Google OAuth popups, set <code>GOOGLE_CLIENT_ID</code> in your <code>.env</code> file.</em>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;

        modalOverlay.style.display = 'flex';

        if (state.authConfig?.google_client_id && window.google?.accounts?.id) {
            const slot = $('modal-gsi-slot');
            if (slot) {
                google.accounts.id.renderButton(slot, {
                    theme: 'filled_black',
                    size: 'large',
                    shape: 'pill',
                    text: 'signin_with',
                });
            }
        }

        $('btn-demo-signin')?.addEventListener('click', async () => {
            try {
                showToast('Signing in with demo RM credentials...', 'info');
                const data = await API.authGoogle('demo:ahmad.mutairi@warbabank.com:Ahmad Al-Mutairi');
                state.currentUser = data.user;
                updateUserUI();
                showToast(`Welcome, ${data.user.name}!`, 'success');
                modalOverlay.style.display = 'none';
            } catch (err) {
                showToast('Demo login error: ' + err.message, 'error');
            }
        });
    }

    // --- Dashboard ---
    async function renderDashboard() {
        const c = container();
        showLoading(c, 'Loading dashboard...');

        try {
            const [summary, opps, clients] = await Promise.all([
                API.portfolioSummary(),
                API.opportunities({}),
                API.clients(),
            ]);
            state.summary = summary;
            state.opportunities = opps;
            state.clients = clients;

            updateOppBadge(summary.new_opportunities);

            c.innerHTML = `
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-label">Total Clients</div>
                        <div class="stat-value blue">${summary.total_clients}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Active Opportunities</div>
                        <div class="stat-value gold">${summary.total_opportunities}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">New (Unreviewed)</div>
                        <div class="stat-value green">${summary.new_opportunities}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Pipeline Value</div>
                        <div class="stat-value gold">${formatKWD(summary.pipeline_value_kwd || 0)}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Avg Confidence</div>
                        <div class="stat-value">${summary.avg_confidence ? (summary.avg_confidence * 100).toFixed(0) + '%' : '—'}</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-label">Converted</div>
                        <div class="stat-value green">${summary.converted_opportunities || 0}</div>
                    </div>
                </div>

                <div class="dashboard-grid">
                    <div class="card">
                        <div class="card-header">
                            <span class="card-title">Urgency Breakdown</span>
                        </div>
                        <div class="chart-container"><canvas id="chart-urgency"></canvas></div>
                    </div>
                    <div class="card">
                        <div class="card-header">
                            <span class="card-title">Top Industries</span>
                        </div>
                        <div class="chart-container"><canvas id="chart-industries"></canvas></div>
                    </div>
                    <div class="card full-width">
                        <div class="card-header">
                            <span class="card-title">Recent Opportunities</span>
                            <a href="#opportunities" class="btn btn-ghost btn-sm">View All →</a>
                        </div>
                        <div class="opp-list" id="dashboard-opps"></div>
                    </div>
                </div>
            `;

            const recentOpps = opps.slice(0, 5);
            const oppList = $('dashboard-opps');
            if (recentOpps.length === 0) {
                showEmpty(oppList, 'No opportunities yet', 'Click "Portfolio Scan" to analyze your clients with AI');
            } else {
                oppList.innerHTML = recentOpps.map(renderOppCard).join('');
            }

            renderUrgencyChart(summary.urgency_breakdown || {});
            renderIndustriesChart(summary.top_industries || []);

        } catch (err) {
            c.innerHTML = `<div class="empty-state"><p>Error loading dashboard</p><p class="sub">${err.message}</p></div>`;
        }
    }

    function renderUrgencyChart(data) {
        const canvas = $('chart-urgency');
        if (!canvas) return;
        const labels = ['Critical', 'High', 'Medium', 'Low'];
        const values = labels.map(l => data[l] || 0);
        const colors = ['#ef4444', '#f97316', '#fbbf24', '#34d399'];

        new Chart(canvas, {
            type: 'doughnut',
            data: {
                labels,
                datasets: [{ data: values, backgroundColor: colors, borderWidth: 0, hoverOffset: 8 }],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                cutout: '65%',
                plugins: {
                    legend: { position: 'bottom', labels: { color: '#94a3b8', font: { size: 12 }, padding: 16, usePointStyle: true } },
                },
            },
        });
    }

    function renderIndustriesChart(data) {
        const canvas = $('chart-industries');
        if (!canvas) return;
        const labels = data.map(d => d.industry);
        const values = data.map(d => d.revenue_kwd / 1000000);
        const colors = data.map(d => industryColor(d.industry));

        new Chart(canvas, {
            type: 'bar',
            data: {
                labels,
                datasets: [{ data: values, backgroundColor: colors.map(c => c + '33'), borderColor: colors, borderWidth: 1, borderRadius: 6 }],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                indexAxis: 'y',
                plugins: { legend: { display: false } },
                scales: {
                    x: { grid: { color: 'rgba(255,255,255,0.03)' }, ticks: { color: '#64748b', font: { size: 11 }, callback: v => 'KWD ' + v + 'M' } },
                    y: { grid: { display: false }, ticks: { color: '#94a3b8', font: { size: 11 } } },
                },
            },
        });
    }

    // --- Clients ---
    async function renderClients() {
        const c = container();
        showLoading(c, 'Loading clients...');

        try {
            const clients = await API.clients();
            state.clients = clients;

            c.innerHTML = `
                <div class="filter-bar">
                    <input type="text" id="client-search" placeholder="Search clients by name, industry...">
                    <select id="client-industry-filter">
                        <option value="">All Industries</option>
                        ${[...new Set(clients.map(cl => cl.industry))].sort().map(i => `<option value="${i}">${i}</option>`).join('')}
                    </select>
                    <select id="client-risk-filter">
                        <option value="">All Risk Levels</option>
                        <option value="Low">Low</option>
                        <option value="Medium">Medium</option>
                        <option value="High">High</option>
                    </select>
                </div>
                <div class="client-list" id="client-list"></div>
            `;

            renderClientList(clients);

            $('client-search').addEventListener('input', filterClients);
            $('client-industry-filter').addEventListener('change', filterClients);
            $('client-risk-filter').addEventListener('change', filterClients);

        } catch (err) {
            c.innerHTML = `<div class="empty-state"><p>Error loading clients</p><p class="sub">${err.message}</p></div>`;
        }
    }

    function filterClients() {
        const search = ($('client-search')?.value || '').toLowerCase();
        const industry = $('client-industry-filter')?.value || '';
        const risk = $('client-risk-filter')?.value || '';

        let filtered = state.clients;
        if (search) filtered = filtered.filter(c => c.name.toLowerCase().includes(search) || c.industry.toLowerCase().includes(search));
        if (industry) filtered = filtered.filter(c => c.industry === industry);
        if (risk) filtered = filtered.filter(c => c.risk_rating === risk);

        renderClientList(filtered);
    }

    function renderClientList(clients) {
        const el = $('client-list');
        if (!el) return;
        if (clients.length === 0) {
            showEmpty(el, 'No clients match your filters');
            return;
        }

        el.innerHTML = clients.map((cl, i) => `
            <div class="client-card" data-id="${cl.id}" onclick="window.location.hash='client/${cl.id}'" style="animation-delay:${i * 0.03}s">
                <div class="client-avatar" style="background:${industryColor(cl.industry)}22; color:${industryColor(cl.industry)}">${initials(cl.name)}</div>
                <div class="client-info">
                    <div class="client-name">${cl.name}</div>
                    <div class="client-meta">
                        <span class="industry-chip industry-${industryClass(cl.industry)}">${cl.industry}</span>
                        <span class="badge risk-${cl.risk_rating.toLowerCase()}">${cl.risk_rating} Risk</span>
                        <span>${cl.employee_count} employees</span>
                    </div>
                </div>
                <div class="client-revenue">${formatKWD(cl.revenue_kwd)}</div>
            </div>
        `).join('');
    }

    // --- Client Detail ---
    async function renderClientDetail(clientId) {
        const c = container();
        showLoading(c, 'Loading client...');

        try {
            const client = await API.client(clientId);

            c.innerHTML = `
                <button class="back-btn" onclick="window.location.hash='clients'">← Back to Clients</button>

                <div class="client-detail-header">
                    <div class="client-detail-avatar" style="background:${industryColor(client.industry)}22; color:${industryColor(client.industry)}">${initials(client.name)}</div>
                    <div class="client-detail-info">
                        <h2>${client.name}</h2>
                        <div class="client-meta">
                            <span class="industry-chip industry-${industryClass(client.industry)}">${client.industry} — ${client.sub_industry}</span>
                            <span class="badge risk-${client.risk_rating.toLowerCase()}">${client.risk_rating} Risk</span>
                            ${client.kyc_status !== 'Active' ? `<span class="badge badge-critical">KYC: ${client.kyc_status}</span>` : ''}
                        </div>
                    </div>
                    <div style="margin-left:auto">
                        <button class="btn btn-primary" id="btn-analyze" onclick="analyzeClient('${client.id}')">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>
                            Analyze with AI
                        </button>
                    </div>
                </div>

                <div class="detail-grid">
                    <div class="detail-item">
                        <div class="detail-label">Revenue</div>
                        <div class="detail-value accent">${formatKWD(client.revenue_kwd)}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">Employees</div>
                        <div class="detail-value">${formatNumber(client.employee_count)}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">Incorporated</div>
                        <div class="detail-value">${client.incorporation_year}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">Relationship Since</div>
                        <div class="detail-value">${client.relationship_start}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">Relationship Manager</div>
                        <div class="detail-value">${client.rm_name || '—'}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">Country</div>
                        <div class="detail-value">${client.country}</div>
                    </div>
                </div>

                ${client.notes ? `<div class="card" style="margin-bottom:20px;padding:16px 20px">
                    <div class="detail-label" style="margin-bottom:8px">Notes</div>
                    <p style="font-size:0.88rem;color:var(--text-secondary);line-height:1.6">${client.notes}</p>
                </div>` : ''}

                <div class="section-header">
                    <span class="section-title">Current Product Holdings</span>
                </div>
                ${renderClientProducts(client.current_products)}

                <div class="section-header">
                    <span class="section-title">Interaction History</span>
                </div>
                ${renderTimeline(client.interactions)}

                <div class="section-header">
                    <span class="section-title">AI-Generated Opportunities</span>
                </div>
                <div id="client-opportunities">
                    <div class="empty-state" style="padding:30px">
                        <p>Click "Analyze with AI" to discover opportunities</p>
                    </div>
                </div>
            `;

            loadClientOpportunities(clientId);

        } catch (err) {
            c.innerHTML = `<div class="empty-state"><p>Error loading client</p><p class="sub">${err.message}</p></div>`;
        }
    }

    function renderClientProducts(products) {
        if (!products || products.length === 0) {
            return '<div class="empty-state" style="padding:30px"><p>No products</p></div>';
        }
        return `<table class="holdings-table">
            <thead><tr>
                <th>Product</th>
                <th>Amount</th>
                <th>Since</th>
                <th>Status</th>
            </tr></thead>
            <tbody>
                ${products.map(p => `<tr>
                    <td style="color:var(--text-primary);font-weight:500">${p.product_name}</td>
                    <td style="color:var(--accent);font-family:var(--font-display);font-weight:600">${formatKWD(p.amount_kwd)}</td>
                    <td>${p.start_date}</td>
                    <td><span class="badge badge-${p.status === 'Active' ? 'accepted' : 'dismissed'}">${p.status}</span></td>
                </tr>`).join('')}
            </tbody>
        </table>`;
    }

    function renderTimeline(interactions) {
        if (!interactions || interactions.length === 0) {
            return '<div class="empty-state" style="padding:30px"><p>No interactions</p></div>';
        }
        return `<div class="timeline">
            ${interactions.map(i => `<div class="timeline-item">
                <div class="timeline-date">${i.date} ${timeAgo(i.date) !== i.date ? '· ' + timeAgo(i.date) : ''}</div>
                <span class="timeline-type ${i.type}">${i.type}</span>
                <div class="timeline-summary">${i.summary}</div>
                ${i.outcome ? `<div class="timeline-outcome">→ ${i.outcome}</div>` : ''}
            </div>`).join('')}
        </div>`;
    }

    async function loadClientOpportunities(clientId) {
        try {
            const opps = await API.opportunities({ client_id: clientId });
            const el = $('client-opportunities');
            if (!el) return;
            if (opps.length === 0) {
                el.innerHTML = '<div class="empty-state" style="padding:30px"><p>No opportunities found</p><p class="sub">Click "Analyze with AI" to generate suggestions</p></div>';
            } else {
                el.innerHTML = '<div class="opp-list">' + opps.map(renderOppCard).join('') + '</div>';
            }
        } catch (err) {
            // Silently fail for initial load
        }
    }

    // --- Analyze Client ---
    window.analyzeClient = async function (clientId) {
        const btn = $('btn-analyze');
        if (btn) {
            btn.disabled = true;
            btn.innerHTML = '<div class="spinner" style="width:16px;height:16px;border-width:2px"></div> Analyzing...';
        }

        try {
            showToast('AI analysis started — this may take a moment...', 'info');
            const opps = await API.analyzeClient(clientId);

            const el = $('client-opportunities');
            if (el && opps.length > 0) {
                el.innerHTML = '<div class="opp-list">' + opps.map(renderOppCard).join('') + '</div>';
                showToast(`Found ${opps.length} opportunity${opps.length > 1 ? 's' : ''}!`, 'success');
            } else if (el) {
                showToast('No new opportunities identified', 'info');
            }
        } catch (err) {
            showToast('Analysis failed: ' + err.message, 'error');
        } finally {
            if (btn) {
                btn.disabled = false;
                btn.innerHTML = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg> Analyze with AI`;
            }
        }
    };

    // --- Opportunities ---
    async function renderOpportunities() {
        const c = container();
        showLoading(c, 'Loading opportunities...');

        try {
            const opps = await API.opportunities({});
            state.opportunities = opps;

            updateOppBadge(opps.filter(o => o.status === 'New').length);

            c.innerHTML = `
                <div class="filter-bar">
                    <input type="text" id="opp-search" placeholder="Search by client, product...">
                    <select id="opp-status-filter">
                        <option value="">All Statuses</option>
                        <option value="New">New</option>
                        <option value="Reviewed">Reviewed</option>
                        <option value="Accepted">Accepted</option>
                        <option value="Dismissed">Dismissed</option>
                        <option value="Converted">Converted</option>
                    </select>
                    <select id="opp-urgency-filter">
                        <option value="">All Urgencies</option>
                        <option value="Critical">Critical</option>
                        <option value="High">High</option>
                        <option value="Medium">Medium</option>
                        <option value="Low">Low</option>
                    </select>
                </div>
                <div class="opp-list" id="opp-list"></div>
            `;

            renderOppList(opps);

            $('opp-search').addEventListener('input', filterOpps);
            $('opp-status-filter').addEventListener('change', filterOpps);
            $('opp-urgency-filter').addEventListener('change', filterOpps);

        } catch (err) {
            c.innerHTML = `<div class="empty-state"><p>Error loading opportunities</p><p class="sub">${err.message}</p></div>`;
        }
    }

    function filterOpps() {
        const search = ($('opp-search')?.value || '').toLowerCase();
        const status = $('opp-status-filter')?.value || '';
        const urgency = $('opp-urgency-filter')?.value || '';

        let filtered = state.opportunities;
        if (search) filtered = filtered.filter(o =>
            (o.client_name || '').toLowerCase().includes(search) ||
            (o.product_name || '').toLowerCase().includes(search) ||
            (o.reasoning || '').toLowerCase().includes(search)
        );
        if (status) filtered = filtered.filter(o => o.status === status);
        if (urgency) filtered = filtered.filter(o => o.urgency === urgency);

        renderOppList(filtered);
    }

    function renderOppList(opps) {
        const el = $('opp-list');
        if (!el) return;
        if (opps.length === 0) {
            showEmpty(el, 'No opportunities found', 'Use "Portfolio Scan" to generate AI-powered suggestions');
            return;
        }
        el.innerHTML = opps.map(renderOppCard).join('');
    }

    function renderOppCard(opp) {
        const urgClass = 'urgency-' + (opp.urgency || 'Medium');
        return `<div class="opp-card ${urgClass}">
            <div class="opp-header">
                <div class="opp-title-group">
                    <div class="opp-client-name">
                        <a href="#client/${opp.client_id}" style="color:inherit;text-decoration:none">${opp.client_name || opp.client_id}</a>
                    </div>
                    <div class="opp-product-name">${opp.product_name || opp.product_id}</div>
                </div>
                <div class="opp-meta">
                    ${statusBadge(opp.status)}
                    ${urgencyBadge(opp.urgency)}
                    ${confidenceMeter(opp.confidence)}
                </div>
            </div>
            <div class="opp-reasoning">${opp.reasoning}</div>
            ${opp.next_action ? `<div class="opp-next-action"><strong>Next Action:</strong> ${opp.next_action}</div>` : ''}
            ${opp.shariah_notes ? `<div class="opp-shariah">${opp.shariah_notes}</div>` : ''}
            <div class="opp-actions">
                ${opp.status === 'New' ? `
                    <button class="btn btn-sm btn-success" onclick="updateOppStatus('${opp.id}', 'Accepted', this)">✓ Accept</button>
                    <button class="btn btn-sm btn-secondary" onclick="updateOppStatus('${opp.id}', 'Reviewed', this)">Mark Reviewed</button>
                    <button class="btn btn-sm btn-danger" onclick="updateOppStatus('${opp.id}', 'Dismissed', this)">✕ Dismiss</button>
                ` : opp.status === 'Accepted' ? `
                    <button class="btn btn-sm btn-primary" onclick="updateOppStatus('${opp.id}', 'Converted', this)">Mark Converted</button>
                    <button class="btn btn-sm btn-danger" onclick="updateOppStatus('${opp.id}', 'Dismissed', this)">✕ Dismiss</button>
                ` : opp.status === 'Reviewed' ? `
                    <button class="btn btn-sm btn-success" onclick="updateOppStatus('${opp.id}', 'Accepted', this)">✓ Accept</button>
                    <button class="btn btn-sm btn-danger" onclick="updateOppStatus('${opp.id}', 'Dismissed', this)">✕ Dismiss</button>
                ` : ''}
                <span style="font-size:0.72rem;color:var(--text-muted);margin-left:auto">${timeAgo(opp.created_at)}</span>
            </div>
        </div>`;
    }

    window.updateOppStatus = async function (id, status, btn) {
        try {
            if (btn) btn.disabled = true;
            await API.updateOppStatus(id, status);
            showToast(`Opportunity ${status.toLowerCase()}`, 'success');

            if (state.currentPage === 'opportunities') {
                renderOpportunities();
            } else if (state.currentPage === 'dashboard') {
                renderDashboard();
            } else if (state.currentPage === 'client') {
                const hash = window.location.hash.slice(1);
                const clientId = hash.split('/')[1];
                if (clientId) loadClientOpportunities(clientId);
            }
        } catch (err) {
            showToast('Failed to update: ' + err.message, 'error');
            if (btn) btn.disabled = false;
        }
    };

    // --- Products ---
    async function renderProducts() {
        const c = container();
        showLoading(c, 'Loading products...');

        try {
            const products = await API.products();
            state.products = products;

            c.innerHTML = `
                <div class="products-grid">
                    ${products.map(p => `
                        <div class="product-card">
                            <div class="product-category">${p.category}</div>
                            <div class="product-name">${p.name}</div>
                            ${p.name_ar ? `<div class="product-name-ar">${p.name_ar}</div>` : ''}
                            <div class="product-description">${p.description}</div>
                            <div class="product-details">
                                <span class="product-tag">📜 ${p.shariah_structure}</span>
                                <span class="product-tag">💰 ${formatKWD(p.min_amount_kwd)} – ${formatKWD(p.max_amount_kwd)}</span>
                                <span class="product-tag">📅 ${p.typical_tenure_months} months</span>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
        } catch (err) {
            c.innerHTML = `<div class="empty-state"><p>Error loading products</p><p class="sub">${err.message}</p></div>`;
        }
    }

    // --- Portfolio Scan ---
    $('btn-portfolio-scan').addEventListener('click', async () => {
        const btn = $('btn-portfolio-scan');
        btn.disabled = true;
        btn.innerHTML = '<div class="spinner" style="width:16px;height:16px;border-width:2px"></div> Scanning...';

        try {
            showToast('AI is scanning your entire portfolio — this may take 30-60 seconds...', 'info');
            const opps = await API.portfolioScan();

            showToast(`Portfolio scan complete! Found ${opps.length} opportunities.`, 'success');

            window.location.hash = 'opportunities';

        } catch (err) {
            showToast('Portfolio scan failed: ' + err.message, 'error');
        } finally {
            btn.disabled = false;
            btn.innerHTML = `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"></path><circle cx="12" cy="12" r="3"></circle></svg> Portfolio Scan`;
        }
    });

    // --- Utility ---
    function updateOppBadge(count) {
        const badge = $('opp-badge');
        if (badge) {
            badge.textContent = count;
            badge.style.display = count > 0 ? 'inline-block' : 'none';
        }
    }

    // --- Modal ---
    $('modal-close').addEventListener('click', () => {
        $('modal-overlay').style.display = 'none';
    });
    $('modal-overlay').addEventListener('click', (e) => {
        if (e.target === $('modal-overlay')) {
            $('modal-overlay').style.display = 'none';
        }
    });

    // --- Init ---
    initAuth();
    initRouter();
})();
