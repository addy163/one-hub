import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Box,
  Button,
  Card,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  IconButton,
  MenuItem,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TablePagination,
  TableRow,
  TextField,
  Tooltip,
  Typography
} from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { Icon } from '@iconify/react';
import dayjs from 'dayjs';

import { API } from 'utils/api';
import { showError, showSuccess } from 'utils/common';
import KeywordTableHead from 'ui-component/TableHead';
import { levelColors, levelMeta } from 'ui-component/notice/NoticeContent';
import NoticeEditModal from './component/NoticeEditModal';

const STATUS_FILTERS = [
  { value: 'all', label: '全部' },
  { value: '1', label: '已发布' },
  { value: '0', label: '草稿' },
  { value: '2', label: '已下线' }
];

function formatRange(notice) {
  const { start_time: start, end_time: end } = notice;
  if (!start && !end) return '长期有效';
  if (start && !end) return `${dayjs(start * 1000).format('MM-DD HH:mm')} 起`;
  if (!start && end) return `至 ${dayjs(end * 1000).format('MM-DD HH:mm')}`;
  return `${dayjs(start * 1000).format('MM-DD HH:mm')} → ${dayjs(end * 1000).format('MM-DD HH:mm')}`;
}

// 综合状态与时间窗，给出一眼能懂的展示状态。
function displayState(notice) {
  if (notice.status === 0) return { label: '草稿', color: 'default' };
  if (notice.status === 2) return { label: '已下线', color: 'default' };
  const now = Date.now() / 1000;
  if (notice.start_time > 0 && notice.start_time > now) return { label: '待生效', color: 'warning' };
  if (notice.end_time > 0 && notice.end_time < now) return { label: '已过期', color: 'default' };
  return { label: '生效中', color: 'success' };
}

function LevelTag({ level }) {
  const theme = useTheme();
  const meta = levelMeta(level);
  const colors = levelColors(theme, level);
  return (
    <Chip
      size="small"
      icon={<Icon icon={meta.icon} width={13} height={13} style={{ color: colors.main }} />}
      label={meta.label}
      sx={{ height: 22, fontSize: '0.75rem', bgcolor: colors.bg, color: colors.fg, '& .MuiChip-icon': { ml: 0.75 } }}
    />
  );
}

export default function Notice() {
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);

  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [deleting, setDeleting] = useState(null);

  const fetchNotices = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/notices/manage/', { params: { page: 1, size: 100, status: 'all' } });
      if (res.data.success) {
        setRows(res.data.data?.data || []);
      } else {
        showError(res.data.message || '加载公告失败');
      }
    } catch (error) {
      showError('加载公告失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchNotices();
  }, [fetchNotices]);

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    return rows.filter((row) => {
      if (statusFilter !== 'all' && String(row.status) !== statusFilter) return false;
      if (!kw) return true;
      return row.title.toLowerCase().includes(kw) || (row.content || '').toLowerCase().includes(kw);
    });
  }, [rows, keyword, statusFilter]);

  const paged = useMemo(
    () => filtered.slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage),
    [filtered, page, rowsPerPage]
  );

  const openCreate = () => {
    setEditing(null);
    setEditOpen(true);
  };

  const openEdit = (row) => {
    setEditing(row);
    setEditOpen(true);
  };

  const changeStatus = async (row, status) => {
    try {
      const res = await API.post(`/api/notices/manage/${row.id}/status`, { status });
      if (res.data.success) {
        showSuccess(status === 1 ? '已发布' : status === 2 ? '已下线' : '已退回草稿');
        fetchNotices();
      } else {
        showError(res.data.message || '操作失败');
      }
    } catch (error) {
      showError('操作失败');
    }
  };

  const confirmDelete = async () => {
    if (!deleting) return;
    try {
      const res = await API.delete(`/api/notices/manage/${deleting.id}`);
      if (res.data.success) {
        showSuccess('公告已删除');
        setDeleting(null);
        fetchNotices();
      } else {
        showError(res.data.message || '删除失败');
      }
    } catch (error) {
      showError('删除失败');
    }
  };

  return (
    <>
      <Card sx={{ borderRadius: '12px' }}>
        <Box sx={{ p: 2.5, display: 'flex', flexWrap: 'wrap', gap: 1.5, alignItems: 'center' }}>
          <Box sx={{ flex: 1, minWidth: 160 }}>
            <Typography variant="h4" sx={{ fontWeight: 600 }}>
              公告管理
            </Typography>
            <Typography variant="caption" sx={{ color: 'text.secondary' }}>
              发布后按设置出现在顶部横幅、铃铛收件箱与首次弹窗
            </Typography>
          </Box>
          <TextField
            size="small"
            placeholder="搜索标题或正文"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value);
              setPage(0);
            }}
            sx={{ width: 200 }}
            InputProps={{
              startAdornment: <Icon icon="solar:magnifer-linear" width={16} height={16} style={{ marginRight: 8, opacity: 0.5 }} />
            }}
          />
          <TextField
            select
            size="small"
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(0);
            }}
            sx={{ width: 120 }}
          >
            {STATUS_FILTERS.map((item) => (
              <MenuItem key={item.value} value={item.value}>
                {item.label}
              </MenuItem>
            ))}
          </TextField>
          <Tooltip title="刷新">
            <IconButton onClick={fetchNotices} size="small">
              <Icon icon="solar:refresh-linear" width={18} height={18} />
            </IconButton>
          </Tooltip>
          <Button variant="contained" disableElevation onClick={openCreate} sx={{ borderRadius: '8px' }} startIcon={<Icon icon="solar:add-circle-bold" width={18} height={18} />}>
            新建公告
          </Button>
        </Box>
        <Divider />

        <TableContainer>
          <Table sx={{ minWidth: 900 }}>
            <KeywordTableHead
              headLabel={[
                { id: 'title', label: '标题', disableSort: true, align: 'left' },
                { id: 'level', label: '级别', disableSort: true },
                { id: 'state', label: '状态', disableSort: true },
                { id: 'channel', label: '展示方式', disableSort: true },
                { id: 'range', label: '生效时间', disableSort: true },
                { id: 'read', label: '已读 / 确认', disableSort: true },
                { id: 'action', label: '操作', disableSort: true }
              ]}
            />
            <TableBody>
              {paged.map((row) => {
                const state = displayState(row);
                return (
                  <TableRow hover key={row.id}>
                    <TableCell align="left" sx={{ maxWidth: 260 }}>
                      <Stack direction="row" spacing={0.75} alignItems="center">
                        {row.pinned && (
                          <Tooltip title="置顶">
                            <Box sx={{ display: 'flex', color: 'warning.main' }}>
                              <Icon icon="solar:pin-bold" width={14} height={14} />
                            </Box>
                          </Tooltip>
                        )}
                        <Typography variant="body2" sx={{ fontWeight: 500 }} noWrap>
                          {row.title}
                        </Typography>
                      </Stack>
                    </TableCell>
                    <TableCell align="right">
                      <LevelTag level={row.level} />
                    </TableCell>
                    <TableCell align="right">
                      <Chip size="small" label={state.label} color={state.color} variant="outlined" sx={{ height: 22, fontSize: '0.75rem' }} />
                    </TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={0.75} justifyContent="flex-end" sx={{ color: 'text.disabled' }}>
                        {row.show_banner && <Tooltip title="顶部横幅"><Box sx={{ display: 'flex' }}><Icon icon="solar:maximize-square-2-bold" width={15} height={15} /></Box></Tooltip>}
                        {row.popup && <Tooltip title="首次弹窗"><Box sx={{ display: 'flex' }}><Icon icon="solar:window-frame-bold" width={15} height={15} /></Box></Tooltip>}
                        {row.require_ack && <Tooltip title="必须确认"><Box sx={{ display: 'flex', color: 'warning.main' }}><Icon icon="solar:shield-check-bold" width={15} height={15} /></Box></Tooltip>}
                        {!row.show_banner && !row.popup && <Typography variant="caption" sx={{ color: 'text.disabled' }}>仅列表</Typography>}
                      </Stack>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        {formatRange(row)}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                        {row.require_ack ? `${row.read_count} / ${row.ack_count}` : row.read_count}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={0.5} justifyContent="flex-end">
                        {row.status === 1 ? (
                          <Tooltip title="下线">
                            <IconButton size="small" onClick={() => changeStatus(row, 2)}>
                              <Icon icon="solar:pause-circle-linear" width={17} height={17} />
                            </IconButton>
                          </Tooltip>
                        ) : (
                          <Tooltip title="发布">
                            <IconButton size="small" onClick={() => changeStatus(row, 1)}>
                              <Icon icon="solar:play-circle-linear" width={17} height={17} />
                            </IconButton>
                          </Tooltip>
                        )}
                        <Tooltip title="编辑">
                          <IconButton size="small" onClick={() => openEdit(row)}>
                            <Icon icon="solar:pen-linear" width={17} height={17} />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="删除">
                          <IconButton size="small" onClick={() => setDeleting(row)}>
                            <Icon icon="solar:trash-bin-trash-linear" width={17} height={17} />
                          </IconButton>
                        </Tooltip>
                      </Stack>
                    </TableCell>
                  </TableRow>
                );
              })}
              {paged.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} align="center" sx={{ py: 8, color: 'text.disabled' }}>
                    <Stack alignItems="center" spacing={1}>
                      <Icon icon="solar:document-add-bold-duotone" width={40} height={40} />
                      <Typography variant="body2">{loading ? '加载中…' : keyword ? '没有匹配的公告' : '还没有公告，点右上角新建一条'}</Typography>
                    </Stack>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>

        <TablePagination
          component="div"
          count={filtered.length}
          page={page}
          rowsPerPage={rowsPerPage}
          onPageChange={(event, value) => setPage(value)}
          rowsPerPageOptions={[10, 20, 50]}
          onRowsPerPageChange={(event) => {
            setRowsPerPage(parseInt(event.target.value, 10));
            setPage(0);
          }}
          labelRowsPerPage="每页"
        />
      </Card>

      <NoticeEditModal
        open={editOpen}
        notice={editing}
        onClose={() => setEditOpen(false)}
        onSaved={() => {
          fetchNotices();
        }}
      />

      <Dialog open={!!deleting} onClose={() => setDeleting(null)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 600 }}>删除公告</DialogTitle>
        <DialogContent>
          <Typography variant="body2" sx={{ color: 'text.secondary' }}>
            确定删除「{deleting?.title}」？删除后用户端立即不可见，已读记录会一并失效。
          </Typography>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2 }}>
          <Button onClick={() => setDeleting(null)} sx={{ color: 'text.secondary' }}>
            取消
          </Button>
          <Button color="error" variant="contained" disableElevation onClick={confirmDelete}>
            删除
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
