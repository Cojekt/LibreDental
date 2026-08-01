<script lang="ts">
  import { onMount } from 'svelte';
  import { PatientService } from "../bindings/github.com/LibreDental/libredental/pkg/services";
  import type { Patient } from "../bindings/github.com/LibreDental/libredental/pkg/domain/models";

  let patients = $state<Patient[]>([]);
  let searchQuery = $state('');
  let loading = $state(false);
  let showAddModal = $state(false);

  // New patient form fields
  let firstName = $state('');
  let lastName = $state('');
  let email = $state('');
  let phone = $state('');
  let dob = $state('1990-01-01');
  let medicalAlerts = $state('Penicillin Allergy');

  async function loadPatients() {
    loading = true;
    try {
      const res = await PatientService.ListPatients(searchQuery);
      patients = res || [];
    } catch (err) {
      console.error("Failed to load patients:", err);
    } finally {
      loading = false;
    }
  }

  async function handleCreatePatient(e: Event) {
    e.preventDefault();
    if (!firstName || !lastName) return;

    try {
      const newPatient: Patient = {
        id: "pat_" + Date.now(),
        first_name: firstName,
        last_name: lastName,
        email: email,
        phone_primary: phone,
        date_of_birth: new Date(dob).toISOString(),
        gender: "undisclosed",
        medical_alerts: medicalAlerts ? medicalAlerts.split(',').map(s => s.trim()) : [],
        allergies: [],
        version: 1,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };

      await PatientService.CreatePatient(newPatient);
      showAddModal = false;
      // Reset form
      firstName = '';
      lastName = '';
      email = '';
      phone = '';
      medicalAlerts = '';
      
      await loadPatients();
    } catch (err) {
      console.error("Failed to create patient:", err);
    }
  }

  onMount(() => {
    loadPatients();
  });
</script>

<div class="app-container">
  <!-- Top Navigation Header -->
  <header class="header">
    <div class="brand">
      <div class="logo">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 14.5v-9l6 4.5-6 4.5z"/>
        </svg>
      </div>
      <div>
        <h1 class="brand-title">LibreDental<span class="tm">™</span></h1>
        <span class="status-pill">Local-First Engine</span>
      </div>
    </div>

    <div class="header-actions">
      <button class="btn btn-primary" onclick={() => showAddModal = true}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        New Patient
      </button>
    </div>
  </header>

  <!-- Main Content Layout -->
  <main class="main-content">
    <!-- Stat Cards -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon text-indigo">👥</div>
        <div>
          <div class="stat-label">Total Patients</div>
          <div class="stat-value">{patients.length}</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon text-teal">📅</div>
        <div>
          <div class="stat-label">Today's Schedule</div>
          <div class="stat-value">0 Appointments</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon text-emerald">🛡️</div>
        <div>
          <div class="stat-label">HIPAA Security</div>
          <div class="stat-value text-emerald">Encrypted Local WAL</div>
        </div>
      </div>
    </div>

    <!-- Search & Filter Controls -->
    <div class="filter-bar">
      <div class="search-box">
        <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <input 
          type="text" 
          placeholder="Search patients by name, email, or phone..." 
          bind:value={searchQuery}
          oninput={loadPatients}
        />
      </div>
    </div>

    <!-- Patient Roster Table -->
    <div class="table-card">
      {#if loading}
        <div class="empty-state">Loading local patient database...</div>
      {:else if patients.length === 0}
        <div class="empty-state">
          <p class="empty-title">No patients found</p>
          <p class="empty-sub">Add your first patient to begin using your local-first LibreDental database.</p>
          <button class="btn btn-secondary" onclick={() => showAddModal = true}>Add First Patient</button>
        </div>
      {:else}
        <table class="data-table">
          <thead>
            <tr>
              <th>Patient Name</th>
              <th>Contact Info</th>
              <th>Date of Birth</th>
              <th>Medical Alerts</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each patients as p}
              <tr>
                <td>
                  <div class="patient-name">{p.first_name} {p.last_name}</div>
                  <div class="patient-id">{p.id}</div>
                </td>
                <td>
                  <div>{p.phone_primary || 'No phone'}</div>
                  <div class="sub-text">{p.email || 'No email'}</div>
                </td>
                <td>{p.date_of_birth ? new Date(p.date_of_birth).toLocaleDateString() : 'N/A'}</td>
                <td>
                  {#if p.medical_alerts && p.medical_alerts.length > 0}
                    <div class="badge-group">
                      {#each p.medical_alerts as alert}
                        <span class="badge badge-warning">⚠️ {alert}</span>
                      {/each}
                    </div>
                  {:else}
                    <span class="badge badge-success">Clean Record</span>
                  {/if}
                </td>
                <td>
                  <button class="btn-sm btn-ghost">Open Chart</button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  </main>
</div>

<!-- Add Patient Modal -->
{#if showAddModal}
  <div class="modal-backdrop" onclick={() => showAddModal = false}>
    <div class="modal-card" onclick={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2>Add New Patient</h2>
        <button class="close-btn" onclick={() => showAddModal = false}>&times;</button>
      </div>

      <form onsubmit={handleCreatePatient}>
        <div class="form-grid">
          <div class="form-group">
            <label for="fname">First Name *</label>
            <input id="fname" type="text" required bind:value={firstName} placeholder="Jane" />
          </div>

          <div class="form-group">
            <label for="lname">Last Name *</label>
            <input id="lname" type="text" required bind:value={lastName} placeholder="Smith" />
          </div>

          <div class="form-group">
            <label for="email">Email</label>
            <input id="email" type="email" bind:value={email} placeholder="jane.smith@example.com" />
          </div>

          <div class="form-group">
            <label for="phone">Phone Primary</label>
            <input id="phone" type="tel" bind:value={phone} placeholder="(555) 019-2834" />
          </div>

          <div class="form-group">
            <label for="dob">Date of Birth</label>
            <input id="dob" type="date" bind:value={dob} />
          </div>

          <div class="form-group">
            <label for="alerts">Medical Alerts (comma separated)</label>
            <input id="alerts" type="text" bind:value={medicalAlerts} placeholder="e.g. Penicillin, Latex, Latex Allergy" />
          </div>
        </div>

        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" onclick={() => showAddModal = false}>Cancel</button>
          <button type="submit" class="btn btn-primary">Save Patient Record</button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: 'Inter', system-ui, -apple-system, sans-serif;
    background-color: #0f172a;
    color: #f8fafc;
  }

  .app-container {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .header {
    height: 64px;
    padding: 0 24px;
    background-color: #1e293b;
    border-bottom: 1px solid #334155;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .logo {
    width: 36px;
    height: 36px;
    background: linear-gradient(135deg, #3b82f6, #06b6d4);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
  }

  .logo svg {
    width: 22px;
    height: 22px;
  }

  .brand-title {
    font-size: 20px;
    font-weight: 700;
    margin: 0;
    letter-spacing: -0.02em;
    color: #f8fafc;
  }

  .tm {
    font-size: 11px;
    color: #94a3b8;
    vertical-align: super;
  }

  .status-pill {
    font-size: 11px;
    background-color: rgba(6, 182, 212, 0.15);
    color: #38bdf8;
    padding: 2px 8px;
    border-radius: 12px;
    border: 1px solid rgba(56, 189, 248, 0.3);
    margin-left: 8px;
  }

  .main-content {
    padding: 24px;
    max-width: 1200px;
    margin: 0 auto;
    width: 100%;
    box-sizing: border-box;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 16px;
    margin-bottom: 24px;
  }

  .stat-card {
    background-color: #1e293b;
    border: 1px solid #334155;
    border-radius: 12px;
    padding: 20px;
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .stat-icon {
    font-size: 28px;
  }

  .stat-label {
    font-size: 13px;
    color: #94a3b8;
  }

  .stat-value {
    font-size: 20px;
    font-weight: 700;
    color: #f8fafc;
    margin-top: 2px;
  }

  .text-emerald { color: #34d399; }

  .filter-bar {
    margin-bottom: 16px;
  }

  .search-box {
    position: relative;
    max-width: 480px;
  }

  .search-icon {
    position: absolute;
    left: 12px;
    top: 50%;
    transform: translateY(-50%);
    width: 18px;
    height: 18px;
    color: #64748b;
  }

  .search-box input {
    width: 100%;
    padding: 10px 12px 10px 40px;
    background-color: #1e293b;
    border: 1px solid #334155;
    border-radius: 8px;
    color: white;
    font-size: 14px;
    box-sizing: border-box;
  }

  .search-box input:focus {
    outline: none;
    border-color: #3b82f6;
  }

  .table-card {
    background-color: #1e293b;
    border: 1px solid #334155;
    border-radius: 12px;
    overflow: hidden;
  }

  .data-table {
    width: 100%;
    border-collapse: collapse;
    text-align: left;
  }

  .data-table th {
    background-color: #0f172a;
    padding: 14px 20px;
    font-size: 12px;
    font-weight: 600;
    color: #94a3b8;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid #334155;
  }

  .data-table td {
    padding: 16px 20px;
    border-bottom: 1px solid #334155;
    font-size: 14px;
  }

  .patient-name {
    font-weight: 600;
    color: #f8fafc;
  }

  .patient-id {
    font-size: 11px;
    color: #64748b;
    margin-top: 2px;
  }

  .sub-text {
    font-size: 12px;
    color: #94a3b8;
  }

  .badge-group {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }

  .badge {
    font-size: 12px;
    padding: 4px 8px;
    border-radius: 6px;
    font-weight: 500;
  }

  .badge-warning {
    background-color: rgba(245, 158, 11, 0.15);
    color: #fbbf24;
    border: 1px solid rgba(251, 191, 36, 0.3);
  }

  .badge-success {
    background-color: rgba(16, 185, 129, 0.15);
    color: #34d399;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    border: none;
    transition: all 0.2s;
  }

  .btn-primary {
    background-color: #2563eb;
    color: white;
  }
  .btn-primary:hover { background-color: #1d4ed8; }

  .btn-secondary {
    background-color: #334155;
    color: #f8fafc;
  }
  .btn-secondary:hover { background-color: #475569; }

  .btn-ghost {
    background: transparent;
    color: #38bdf8;
    border: 1px solid rgba(56, 189, 248, 0.3);
    padding: 6px 12px;
    font-size: 12px;
    border-radius: 6px;
    cursor: pointer;
  }
  .btn-ghost:hover { background: rgba(56, 189, 248, 0.1); }

  .empty-state {
    padding: 48px;
    text-align: center;
    color: #94a3b8;
  }

  .empty-title {
    font-size: 18px;
    font-weight: 600;
    color: #f8fafc;
    margin-bottom: 8px;
  }

  .empty-sub {
    font-size: 14px;
    margin-bottom: 16px;
  }

  /* Modal Styling */
  .modal-backdrop {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background-color: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal-card {
    background-color: #1e293b;
    border: 1px solid #334155;
    border-radius: 12px;
    width: 100%;
    max-width: 540px;
    padding: 24px;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }

  .modal-header h2 {
    margin: 0;
    font-size: 18px;
    color: #f8fafc;
  }

  .close-btn {
    background: none;
    border: none;
    color: #94a3b8;
    font-size: 24px;
    cursor: pointer;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
    margin-bottom: 24px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .form-group label {
    font-size: 12px;
    color: #94a3b8;
    font-weight: 500;
  }

  .form-group input {
    padding: 8px 12px;
    background-color: #0f172a;
    border: 1px solid #334155;
    border-radius: 6px;
    color: white;
    font-size: 14px;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
</style>
