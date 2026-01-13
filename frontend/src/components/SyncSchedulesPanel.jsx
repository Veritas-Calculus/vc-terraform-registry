import { useState, useEffect, useCallback } from 'react';
import {
  fetchSyncSchedules,
  createSyncSchedule,
  updateSyncSchedule,
  deleteSyncSchedule,
  runSyncScheduleNow
} from '../services/api';

export default function SyncSchedulesPanel({ onMessage }) {
  const [schedules, setSchedules] = useState([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({
    namespace: '',
    name: '',
    cronExpr: '0 0 * * *',
    syncOS: 'all',
    syncArch: 'all',
    enabled: true
  });
  const [editingId, setEditingId] = useState(null);
  const [showForm, setShowForm] = useState(false);

  const loadSchedules = useCallback(async () => {
    try {
      setLoading(true);
      const data = await fetchSyncSchedules();
      setSchedules(data.schedules || []);
    } catch (err) {
      onMessage({ type: 'error', text: 'Failed to load schedules: ' + err.message });
    } finally {
      setLoading(false);
    }
  }, [onMessage]);

  useEffect(() => {
    loadSchedules();
  }, [loadSchedules]);

  const resetForm = () => {
    setForm({
      namespace: '',
      name: '',
      cronExpr: '0 0 * * *',
      syncOS: 'all',
      syncArch: 'all',
      enabled: true
    });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      if (editingId) {
        await updateSyncSchedule(editingId, {
          cron_expr: form.cronExpr,
          sync_os: form.syncOS,
          sync_arch: form.syncArch,
          enabled: form.enabled
        });
        onMessage({ type: 'success', text: 'Schedule updated successfully' });
      } else {
        await createSyncSchedule({
          namespace: form.namespace,
          name: form.name,
          cron_expr: form.cronExpr,
          sync_os: form.syncOS,
          sync_arch: form.syncArch,
          enabled: form.enabled
        });
        onMessage({ type: 'success', text: 'Schedule created successfully' });
      }
      resetForm();
      loadSchedules();
    } catch (err) {
      onMessage({ type: 'error', text: 'Failed to save schedule: ' + err.message });
    }
  };

  const handleEdit = (schedule) => {
    setForm({
      namespace: schedule.namespace,
      name: schedule.name,
      cronExpr: schedule.cron_expr,
      syncOS: schedule.sync_os || 'all',
      syncArch: schedule.sync_arch || 'all',
      enabled: schedule.enabled
    });
    setEditingId(schedule.ID);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!window.confirm('Are you sure you want to delete this schedule?')) {
      return;
    }
    try {
      await deleteSyncSchedule(id);
      onMessage({ type: 'success', text: 'Schedule deleted successfully' });
      loadSchedules();
    } catch (err) {
      onMessage({ type: 'error', text: 'Failed to delete schedule: ' + err.message });
    }
  };

  const handleRunNow = async (id) => {
    try {
      await runSyncScheduleNow(id);
      onMessage({ type: 'success', text: 'Sync triggered successfully' });
      // Reload after a short delay to show updated status
      setTimeout(loadSchedules, 2000);
    } catch (err) {
      onMessage({ type: 'error', text: 'Failed to trigger sync: ' + err.message });
    }
  };

  const handleToggleEnabled = async (schedule) => {
    try {
      await updateSyncSchedule(schedule.ID, {
        enabled: !schedule.enabled
      });
      loadSchedules();
    } catch (err) {
      onMessage({ type: 'error', text: 'Failed to update schedule: ' + err.message });
    }
  };

  const formatDate = (dateStr) => {
    if (!dateStr || dateStr === '0001-01-01T00:00:00Z') {
      return '-';
    }
    return new Date(dateStr).toLocaleString();
  };

  const cronPresets = [
    { label: 'Every hour', value: '0 * * * *' },
    { label: 'Every 6 hours', value: '0 */6 * * *' },
    { label: 'Daily at midnight', value: '0 0 * * *' },
    { label: 'Weekly on Sunday', value: '0 0 * * 0' },
    { label: 'Monthly', value: '0 0 1 * *' }
  ];

  if (loading) {
    return (
      <div className="flex justify-center items-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">Sync Schedules</h2>
          <p className="text-sm text-gray-500 mt-1">
            Configure automatic synchronization with upstream registry
          </p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          {showForm ? 'Cancel' : 'Add Schedule'}
        </button>
      </div>

      {/* Form */}
      {showForm && (
        <div className="bg-gray-50 rounded-xl p-6 border border-gray-200">
          <h3 className="text-lg font-medium text-gray-900 mb-4">
            {editingId ? 'Edit Schedule' : 'New Schedule'}
          </h3>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Namespace
                </label>
                <input
                  type="text"
                  value={form.namespace}
                  onChange={(e) => setForm({ ...form, namespace: e.target.value })}
                  disabled={!!editingId}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-100 disabled:cursor-not-allowed"
                  placeholder="e.g., hashicorp"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Provider Name
                </label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  disabled={!!editingId}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-100 disabled:cursor-not-allowed"
                  placeholder="e.g., aws"
                  required
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Cron Expression
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={form.cronExpr}
                  onChange={(e) => setForm({ ...form, cronExpr: e.target.value })}
                  className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
                  placeholder="0 0 * * *"
                  required
                />
                <select
                  onChange={(e) => e.target.value && setForm({ ...form, cronExpr: e.target.value })}
                  className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  defaultValue=""
                >
                  <option value="">Presets</option>
                  {cronPresets.map((preset) => (
                    <option key={preset.value} value={preset.value}>
                      {preset.label}
                    </option>
                  ))}
                </select>
              </div>
              <p className="text-xs text-gray-500 mt-1">
                Format: minute hour day month weekday
              </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  OS Filter
                </label>
                <select
                  value={form.syncOS}
                  onChange={(e) => setForm({ ...form, syncOS: e.target.value })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="all">All platforms</option>
                  <option value="linux">Linux</option>
                  <option value="darwin">macOS</option>
                  <option value="windows">Windows</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Architecture Filter
                </label>
                <select
                  value={form.syncArch}
                  onChange={(e) => setForm({ ...form, syncArch: e.target.value })}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="all">All architectures</option>
                  <option value="amd64">amd64</option>
                  <option value="arm64">arm64</option>
                  <option value="386">386</option>
                </select>
              </div>
            </div>

            <div className="flex items-center">
              <input
                type="checkbox"
                id="enabled"
                checked={form.enabled}
                onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="enabled" className="ml-2 block text-sm text-gray-700">
                Enable schedule
              </label>
            </div>

            <div className="flex justify-end gap-3 pt-4">
              <button
                type="button"
                onClick={resetForm}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
              >
                {editingId ? 'Update Schedule' : 'Create Schedule'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Schedule List */}
      {schedules.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-xl">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <h3 className="mt-4 text-lg font-medium text-gray-900">No schedules configured</h3>
          <p className="mt-2 text-gray-500">
            Create a schedule to automatically sync providers from upstream
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {schedules.map((schedule) => (
            <div
              key={schedule.ID}
              className={`bg-white rounded-xl border ${
                schedule.enabled ? 'border-gray-200' : 'border-gray-200 opacity-60'
              } p-4`}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <h4 className="font-medium text-gray-900">
                      {schedule.namespace}/{schedule.name}
                    </h4>
                    <span
                      className={`px-2 py-0.5 text-xs font-medium rounded-full ${
                        schedule.enabled
                          ? 'bg-green-100 text-green-800'
                          : 'bg-gray-100 text-gray-600'
                      }`}
                    >
                      {schedule.enabled ? 'Active' : 'Disabled'}
                    </span>
                    {schedule.last_status && (
                      <span
                        className={`px-2 py-0.5 text-xs font-medium rounded-full ${
                          schedule.last_status === 'success'
                            ? 'bg-blue-100 text-blue-800'
                            : schedule.last_status === 'running'
                            ? 'bg-yellow-100 text-yellow-800'
                            : 'bg-red-100 text-red-800'
                        }`}
                      >
                        {schedule.last_status}
                      </span>
                    )}
                  </div>
                  <div className="mt-2 flex flex-wrap gap-x-6 gap-y-1 text-sm text-gray-500">
                    <span className="font-mono">{schedule.cron_expr}</span>
                    <span>OS: {schedule.sync_os || 'all'}</span>
                    <span>Arch: {schedule.sync_arch || 'all'}</span>
                  </div>
                  <div className="mt-2 flex flex-wrap gap-x-6 gap-y-1 text-xs text-gray-400">
                    <span>Last run: {formatDate(schedule.last_run_at)}</span>
                    <span>Next run: {formatDate(schedule.next_run_at)}</span>
                  </div>
                  {schedule.last_error && (
                    <p className="mt-2 text-sm text-red-600">{schedule.last_error}</p>
                  )}
                </div>
                <div className="flex items-center gap-2 ml-4">
                  <button
                    onClick={() => handleToggleEnabled(schedule)}
                    className={`p-2 rounded-lg transition-colors ${
                      schedule.enabled
                        ? 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'
                        : 'text-green-500 hover:text-green-700 hover:bg-green-50'
                    }`}
                    title={schedule.enabled ? 'Disable' : 'Enable'}
                  >
                    {schedule.enabled ? (
                      <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    ) : (
                      <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    )}
                  </button>
                  <button
                    onClick={() => handleRunNow(schedule.ID)}
                    className="p-2 text-blue-500 hover:text-blue-700 hover:bg-blue-50 rounded-lg transition-colors"
                    title="Run Now"
                  >
                    <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                  </button>
                  <button
                    onClick={() => handleEdit(schedule)}
                    className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
                    title="Edit"
                  >
                    <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                    </svg>
                  </button>
                  <button
                    onClick={() => handleDelete(schedule.ID)}
                    className="p-2 text-red-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                    title="Delete"
                  >
                    <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
