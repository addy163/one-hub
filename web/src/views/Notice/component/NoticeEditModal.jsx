import PropTypes from 'prop-types';
import { useEffect, useMemo, useState } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  Divider,
  FormControlLabel,
  MenuItem,
  Stack,
  Switch,
  TextField,
  Typography
} from '@mui/material';
import { alpha, useTheme } from '@mui/material/styles';
import { Icon } from '@iconify/react';
import dayjs from 'dayjs';
import { marked } from 'marked';

import { API } from 'utils/api';
import { sanitizeHtml } from 'utils/sanitize';
import { showError, showSuccess } from 'utils/common';
import { markdownSx } from 'ui-component/notice/NoticeContent';

const LEVEL_OPTIONS = [
  { value: 'info', label: '通知', icon: 'solar:info-circle-bold' },
  { value: 'success', label: '上新', icon: 'solar:check-circle-bold' },
  { value: 'warning', label: '重要', icon: 'solar:danger-triangle-bold' },
  { value: 'error', label: '紧急', icon: 'solar:danger-circle-bold' }
];

const EMPTY = {
  id: 0,
  title: '',
  content: '',
  level: 'info',
  pinned: false,
  show_banner: false,
  popup: true,
  require_ack: false,
  status: 1,
  start_time: 0,
  end_time: 0
};

function toInputValue(ts) {
  if (!ts) return '';
  return dayjs(ts * 1000).format('YYYY-MM-DDTHH:mm');
}

function toTimestamp(value) {
  if (!value) return 0;
  const t = new Date(value).getTime();
  return Number.isNaN(t) ? 0 : Math.floor(t / 1000);
}

export default function NoticeEditModal({ open, onClose, onSaved, notice }) {
  const theme = useTheme();
  const [form, setForm] = useState(EMPTY);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    setForm(notice ? { ...EMPTY, ...notice } : EMPTY);
  }, [open, notice]);

  const setField = (key, value) => setForm((prev) => ({ ...prev, [key]: value }));

  const preview = useMemo(() => sanitizeHtml(marked(form.content || '').toString()), [form.content]);

  const handleSave = async () => {
    if (!form.title.trim()) {
      showError('请填写公告标题');
      return;
    }
    setSaving(true);
    try {
      const payload = { ...form, title: form.title.trim() };
      const res = form.id > 0 ? await API.put('/api/notices/manage/', payload) : await API.post('/api/notices/manage/', payload);
      if (res.data.success) {
        showSuccess(form.id > 0 ? '公告已更新' : '公告已创建');
        onSaved?.();
        onClose?.();
      } else {
        showError(res.data.message || '保存失败');
      }
    } catch (error) {
      showError(error?.response?.data?.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="lg" PaperProps={{ sx: { borderRadius: '14px' } }}>
      <Box sx={{ px: 3, pt: 2.5, pb: 1.5, display: 'flex', alignItems: 'center', gap: 1 }}>
        <Icon icon="solar:document-text-bold-duotone" width={20} height={20} style={{ opacity: 0.7 }} />
        <Typography variant="h4" sx={{ fontWeight: 600, flex: 1 }}>
          {form.id > 0 ? '编辑公告' : '新建公告'}
        </Typography>
        <TextField
          select
          size="small"
          value={form.status}
          onChange={(e) => setField('status', Number(e.target.value))}
          sx={{ width: 110 }}
        >
          <MenuItem value={0}>草稿</MenuItem>
          <MenuItem value={1}>已发布</MenuItem>
          <MenuItem value={2}>已下线</MenuItem>
        </TextField>
      </Box>
      <Divider />

      <DialogContent sx={{ p: 0 }}>
        <Box sx={{ display: 'flex', flexDirection: { xs: 'column', md: 'row' }, minHeight: 420 }}>
          <Box sx={{ flex: 1, minWidth: 0, p: 3 }}>
            <Stack spacing={2}>
              <TextField
                label="标题"
                size="small"
                fullWidth
                value={form.title}
                onChange={(e) => setField('title', e.target.value)}
                placeholder="例如：11 月 20 日凌晨线路维护"
                inputProps={{ maxLength: 200 }}
              />

              <TextField
                select
                label="级别"
                size="small"
                fullWidth
                value={form.level}
                onChange={(e) => setField('level', e.target.value)}
              >
                {LEVEL_OPTIONS.map((item) => (
                  <MenuItem key={item.value} value={item.value}>
                    {item.label}
                  </MenuItem>
                ))}
              </TextField>

              <TextField
                label="正文（支持 Markdown）"
                multiline
                minRows={12}
                fullWidth
                value={form.content}
                onChange={(e) => setField('content', e.target.value)}
                placeholder={'支持标题、列表、表格、代码块与链接。\n\n- 列表项\n- **加粗** 文字'}
              />

              <Box>
                <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block', mb: 0.5 }}>
                  生效时间（留空表示立即生效 / 长期有效，按本地时区）
                </Typography>
                <Box sx={{ display: 'flex', gap: 1.5 }}>
                  <TextField
                    type="datetime-local"
                    size="small"
                    label="开始"
                    fullWidth
                    value={toInputValue(form.start_time)}
                    onChange={(e) => setField('start_time', toTimestamp(e.target.value))}
                    InputLabelProps={{ shrink: true }}
                  />
                  <TextField
                    type="datetime-local"
                    size="small"
                    label="结束"
                    fullWidth
                    value={toInputValue(form.end_time)}
                    onChange={(e) => setField('end_time', toTimestamp(e.target.value))}
                    InputLabelProps={{ shrink: true }}
                  />
                </Box>
              </Box>

              <Box
                sx={{
                  display: 'grid',
                  gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' },
                  gap: 0.5,
                  p: 1.5,
                  borderRadius: '10px',
                  bgcolor: alpha(theme.palette.primary.main, theme.palette.mode === 'dark' ? 0.08 : 0.04)
                }}
              >
                <FormControlLabel
                  control={<Switch size="small" checked={form.pinned} onChange={(e) => setField('pinned', e.target.checked)} />}
                  label={<Typography variant="body2">置顶（列表优先展示）</Typography>}
                />
                <FormControlLabel
                  control={
                    <Switch size="small" checked={form.show_banner} onChange={(e) => setField('show_banner', e.target.checked)} />
                  }
                  label={<Typography variant="body2">顶部横幅</Typography>}
                />
                <FormControlLabel
                  control={<Switch size="small" checked={form.popup} onChange={(e) => setField('popup', e.target.checked)} />}
                  label={<Typography variant="body2">首次进入弹窗</Typography>}
                />
                <FormControlLabel
                  control={
                    <Switch size="small" checked={form.require_ack} onChange={(e) => setField('require_ack', e.target.checked)} />
                  }
                  label={<Typography variant="body2">必须确认（不可直接关闭）</Typography>}
                />
              </Box>
            </Stack>
          </Box>

          <Divider orientation="vertical" flexItem sx={{ display: { xs: 'none', md: 'block' } }} />

          <Box sx={{ flex: 1, minWidth: 0, p: 3, bgcolor: 'action.hover' }}>
            <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block', mb: 1.5 }}>
              实时预览
            </Typography>
            <Box
              sx={{
                borderRadius: '10px',
                border: '1px solid',
                borderColor: 'divider',
                bgcolor: 'background.paper',
                p: 2.5,
                maxHeight: 440,
                overflowY: 'auto'
              }}
            >
              <Typography variant="h3" sx={{ fontWeight: 600, mb: 1.5, lineHeight: 1.4 }}>
                {form.title || '公告标题'}
              </Typography>
              {form.content ? (
                <Box sx={markdownSx} dangerouslySetInnerHTML={{ __html: preview }} />
              ) : (
                <Typography variant="body2" sx={{ color: 'text.disabled' }}>
                  正文会在这里实时预览。
                </Typography>
              )}
            </Box>
          </Box>
        </Box>
      </DialogContent>

      <Divider />
      <DialogActions sx={{ px: 3, py: 2 }}>
        <Button onClick={onClose} sx={{ color: 'text.secondary' }}>
          取消
        </Button>
        <Button variant="contained" disableElevation onClick={handleSave} disabled={saving} sx={{ borderRadius: '8px', px: 2.5 }}>
          {saving ? '保存中…' : '保存'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

NoticeEditModal.propTypes = {
  open: PropTypes.bool,
  onClose: PropTypes.func,
  onSaved: PropTypes.func,
  notice: PropTypes.object
};
