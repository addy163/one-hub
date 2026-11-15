import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Box, Button, Chip, FormControlLabel, MenuItem, Stack, Switch, TextField, Typography } from '@mui/material';
import Grid from '@mui/material/Unstable_Grid2';
import SubCard from 'ui-component/cards/SubCard';
import { API } from 'utils/api';
import { showError, showSuccess } from 'utils/common';

const GROUP_ORDER = ['passthrough', 'session', 'cors', 'ssrf', 'websocket'];

// 安全 / 透传 / WebSocket 治理配置。
// 表单完全由后端 /api/option/runtime 返回的 schema 驱动，新增配置项无需改前端。
const RuntimeSetting = () => {
  const { t } = useTranslation();
  const tr = (key, options) => t(`setting_index.runtimeSettings.${key}`, options);

  const [specs, setSpecs] = useState([]);
  const [values, setValues] = useState({});
  const [savedValues, setSavedValues] = useState({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/option/runtime');
      if (res.data?.success) {
        const list = res.data.data || [];
        const next = {};
        list.forEach((spec) => {
          next[spec.key] = spec.value ?? '';
        });
        setSpecs(list);
        setValues(next);
        setSavedValues(next);
      } else {
        showError(res.data?.message || t('setting_index.runtimeSettings.actions.loadFailed'));
      }
    } catch (error) {
      // 请求拦截器已经统一提示
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    load().then();
  }, [load]);

  const saveGroup = async (group) => {
    const dirty = specs.filter((spec) => spec.group === group && String(values[spec.key]) !== String(savedValues[spec.key]));
    if (dirty.length === 0) return;
    setSaving(group);
    try {
      for (const spec of dirty) {
        const res = await API.put('/api/option/', { key: spec.key, value: String(values[spec.key]) });
        if (!res.data?.success) {
          showError(`${spec.key}: ${res.data?.message || ''}`);
          await load();
          return;
        }
      }
      showSuccess(tr('actions.saved'));
      await load();
    } finally {
      setSaving('');
    }
  };

  const resetField = async (key) => {
    setSaving(key);
    try {
      const res = await API.delete(`/api/option/runtime/${encodeURIComponent(key)}`);
      if (res.data?.success) {
        showSuccess(tr('actions.resetDone'));
        await load();
      } else {
        showError(res.data?.message || '');
      }
    } finally {
      setSaving('');
    }
  };

  const renderControl = (spec) => {
    const value = values[spec.key] ?? '';
    const onChange = (next) => setValues((prev) => ({ ...prev, [spec.key]: next }));
    const label = tr(`fields.${spec.key}.label`, { defaultValue: spec.key });

    if (spec.type === 'bool') {
      return (
        <FormControlLabel
          control={
            <Switch
              checked={value === 'true'}
              onChange={(event) => onChange(event.target.checked ? 'true' : 'false')}
              disabled={loading}
            />
          }
          label={`${label} · ${value === 'true' ? tr('actions.boolOn') : tr('actions.boolOff')}`}
        />
      );
    }

    if (spec.type === 'enum') {
      return (
        <TextField select fullWidth label={label} value={value} onChange={(event) => onChange(event.target.value)} disabled={loading}>
          {(spec.options || []).map((option) => (
            <MenuItem key={option} value={option}>
              {tr(`actions.mode${option.charAt(0).toUpperCase()}${option.slice(1)}`, { defaultValue: option })}
            </MenuItem>
          ))}
        </TextField>
      );
    }

    return (
      <TextField
        fullWidth
        label={label}
        type={spec.type === 'int' ? 'number' : 'text'}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={spec.default || ''}
        disabled={loading}
        inputProps={spec.type === 'int' ? { min: spec.min, max: spec.max } : undefined}
        helperText={spec.type === 'list' ? tr('actions.listHint') : spec.type === 'int' ? `${spec.min} ~ ${spec.max}` : ''}
      />
    );
  };

  const renderField = (spec) => {
    const dirty = String(values[spec.key]) !== String(savedValues[spec.key]);
    return (
      <Grid key={spec.key} xs={12} md={6}>
        <Box sx={{ p: 2, height: '100%', border: '1px solid', borderColor: 'divider', borderRadius: 2 }}>
          <Stack spacing={1}>
            {renderControl(spec)}
            <Typography variant="caption" color="text.secondary">
              {tr(`fields.${spec.key}.help`, { defaultValue: '' })}
            </Typography>
            <Stack direction="row" spacing={1} flexWrap="wrap" alignItems="center">
              <Chip
                size="small"
                variant="outlined"
                color={spec.source === 'database' ? 'primary' : 'default'}
                label={spec.source === 'database' ? tr('actions.sourceDatabase') : tr('actions.sourceConfig')}
              />
              <Typography variant="caption" color="text.secondary">
                {tr('actions.sourceConfig')}: {spec.default || '-'}
              </Typography>
              {spec.restart_required && <Chip size="small" color="warning" label={tr('actions.restartRequired')} />}
              {dirty && <Chip size="small" color="secondary" label={tr('actions.dirty')} />}
              {spec.source === 'database' && (
                <Button size="small" onClick={() => resetField(spec.key)} disabled={saving === spec.key || loading}>
                  {tr('actions.reset')}
                </Button>
              )}
            </Stack>
          </Stack>
        </Box>
      </Grid>
    );
  };

  return (
    <Stack spacing={2}>
      <Alert severity="info">{tr('subtitle')}</Alert>
      <Stack direction="row" justifyContent="flex-end">
        <Button variant="outlined" onClick={load} disabled={loading || saving !== ''}>
          {tr('actions.reload')}
        </Button>
      </Stack>
      {GROUP_ORDER.map((group) => {
        const list = specs.filter((spec) => spec.group === group);
        if (list.length === 0) return null;
        const dirty = list.some((spec) => String(values[spec.key]) !== String(savedValues[spec.key]));
        return (
          <SubCard key={group} title={tr(`groups.${group}.title`, { defaultValue: group })}>
            <Stack spacing={2}>
              <Typography variant="body2" color="text.secondary">
                {tr(`groups.${group}.description`, { defaultValue: '' })}
              </Typography>
              <Grid container spacing={2}>
                {list.map(renderField)}
              </Grid>
              <Stack direction="row" justifyContent="flex-end">
                <Button
                  variant="contained"
                  onClick={() => saveGroup(group)}
                  disabled={!dirty || loading || saving !== ''}
                >
                  {tr('actions.save')}
                </Button>
              </Stack>
            </Stack>
          </SubCard>
        );
      })}
    </Stack>
  );
};

export default RuntimeSetting;
