import PropTypes from 'prop-types';
import { useState, useEffect, useMemo, useCallback } from 'react';
import { GridRowModes, DataGrid, GridToolbarContainer, GridActionsCellItem } from '@mui/x-data-grid';
import { Box, Button } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/DeleteOutlined';
import SaveIcon from '@mui/icons-material/Save';
import CancelIcon from '@mui/icons-material/Close';
import { showError } from 'utils/common';

function validation(row) {
  if (!row.name) {
    return '线路名称不能为空';
  }
  if (!row.url) {
    return '地址不能为空';
  }
  if (!/^https?:\/\/[^\s]+$/i.test(row.url)) {
    return '地址必须以 http:// 或 https:// 开头';
  }
  if (row.sort !== '' && row.sort != null && !/^-?[0-9]\d*$/.test(String(row.sort))) {
    return '排序必须为整数';
  }
  if (row.groups && !/^[A-Za-z0-9_\-,\s]*$/.test(row.groups)) {
    return '分组只能包含字母、数字、下划线、中划线和逗号';
  }
  return false;
}

function randomId() {
  return Math.random().toString(36).substr(2, 9);
}

function parseGroups(value) {
  if (Array.isArray(value)) return value;
  return String(value || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function toRows(json) {
  let list = [];
  try {
    list = JSON.parse(json || '[]');
  } catch (e) {
    list = [];
  }
  if (!Array.isArray(list)) list = [];

  return list.map((item) => ({
    id: item.id || randomId(),
    name: item.name || '',
    url: item.url || '',
    groups: parseGroups(item.groups).join(','),
    note: item.note || '',
    enabled: item.enabled !== false,
    sort: Number(item.sort) || 0
  }));
}

function toJson(rows) {
  const payload = rows.map((row) => ({
    id: row.id,
    name: row.name,
    url: row.url,
    groups: parseGroups(row.groups),
    note: row.note,
    enabled: row.enabled !== false,
    sort: Number(row.sort) || 0
  }));
  return JSON.stringify(payload, null, 2);
}

function EditToolbar({ setRows, setRowModesModel }) {
  const handleClick = () => {
    const id = randomId();
    setRows((oldRows) => [{ id, name: '', url: '', groups: '', note: '', enabled: true, sort: 0, isNew: true }, ...oldRows]);
    setRowModesModel((oldModel) => ({
      [id]: { mode: GridRowModes.Edit, fieldToFocus: 'name' },
      ...oldModel
    }));
  };

  return (
    <GridToolbarContainer>
      <Button color="primary" startIcon={<AddIcon />} onClick={handleClick}>
        新增线路
      </Button>
    </GridToolbarContainer>
  );
}

EditToolbar.propTypes = {
  setRows: PropTypes.func.isRequired,
  setRowModesModel: PropTypes.func.isRequired
};

const RelayEndpointsDataGrid = ({ endpoints, onChange }) => {
  const [rows, setRows] = useState([]);
  const [rowModesModel, setRowModesModel] = useState({});

  const setEndpoints = useCallback(
    (nextRows) => {
      onChange({ target: { name: 'RelayEndpoints', value: toJson(nextRows) } });
    },
    [onChange]
  );

  const handleEditClick = useCallback(
    (id) => () => {
      setRowModesModel({ ...rowModesModel, [id]: { mode: GridRowModes.Edit } });
    },
    [rowModesModel]
  );

  const handleSaveClick = useCallback(
    (id) => () => {
      setRowModesModel({ ...rowModesModel, [id]: { mode: GridRowModes.View } });
    },
    [rowModesModel]
  );

  const handleDeleteClick = useCallback(
    (id) => () => {
      setEndpoints(rows.filter((row) => row.id !== id));
    },
    [rows, setEndpoints]
  );

  const handleCancelClick = useCallback(
    (id) => () => {
      setRowModesModel({
        ...rowModesModel,
        [id]: { mode: GridRowModes.View, ignoreModifications: true }
      });

      const editedRow = rows.find((row) => row.id === id);
      if (editedRow && editedRow.isNew) {
        setRows(rows.filter((row) => row.id !== id));
      }
    },
    [rowModesModel, rows]
  );

  const processRowUpdate = (newRow, oldRows) => {
    if (
      !newRow.isNew &&
      newRow.name === oldRows.name &&
      newRow.url === oldRows.url &&
      newRow.groups === oldRows.groups &&
      newRow.note === oldRows.note &&
      newRow.enabled === oldRows.enabled &&
      newRow.sort === oldRows.sort
    ) {
      return oldRows;
    }
    const updatedRow = { ...newRow, isNew: false };
    const error = validation(updatedRow);
    if (error) {
      return Promise.reject(new Error(error));
    }
    setEndpoints(rows.map((row) => (row.id === newRow.id ? updatedRow : row)));
    return updatedRow;
  };

  const handleProcessRowUpdateError = useCallback((error) => {
    showError(error.message);
  }, []);

  const handleRowModesModelChange = (newRowModesModel) => {
    setRowModesModel(newRowModesModel);
  };

  const columns = useMemo(
    () => [
      { field: 'name', sortable: true, headerName: '名称', flex: 1, minWidth: 140, editable: true, hideable: false },
      { field: 'url', sortable: false, headerName: '地址', flex: 1, minWidth: 260, editable: true, hideable: false },
      {
        field: 'groups',
        sortable: false,
        headerName: '可见分组',
        flex: 1,
        minWidth: 160,
        editable: true,
        hideable: false,
        description: '逗号分隔，留空表示所有分组可见'
      },
      { field: 'note', sortable: false, headerName: '说明', flex: 1, minWidth: 160, editable: true, hideable: false },
      { field: 'enabled', sortable: false, headerName: '启用', flex: 1, minWidth: 90, type: 'boolean', editable: true, hideable: false },
      { field: 'sort', sortable: true, headerName: '排序', type: 'number', flex: 1, minWidth: 90, editable: true, hideable: false },
      {
        field: 'actions',
        type: 'actions',
        headerName: '操作',
        width: 100,
        cellClassName: 'actions',
        hideable: false,
        getActions: ({ id }) => {
          const isInEditMode = rowModesModel[id]?.mode === GridRowModes.Edit;

          if (isInEditMode) {
            return [
              <GridActionsCellItem icon={<SaveIcon />} key={'Save-' + id} label="Save" sx={{ color: 'primary.main' }} onClick={handleSaveClick(id)} />,
              <GridActionsCellItem
                icon={<CancelIcon />}
                key={'Cancel-' + id}
                label="Cancel"
                className="textPrimary"
                onClick={handleCancelClick(id)}
                color="inherit"
              />
            ];
          }

          return [
            <GridActionsCellItem
              key={'Edit-' + id}
              icon={<EditIcon />}
              label="Edit"
              className="textPrimary"
              onClick={handleEditClick(id)}
              color="inherit"
            />,
            <GridActionsCellItem key={'Delete-' + id} icon={<DeleteIcon />} label="Delete" onClick={handleDeleteClick(id)} color="inherit" />
          ];
        }
      }
    ],
    [handleEditClick, handleSaveClick, handleDeleteClick, handleCancelClick, rowModesModel]
  );

  useEffect(() => {
    setRows(toRows(endpoints));
  }, [endpoints]);

  return (
    <Box
      sx={{
        width: '100%',
        '& .actions': { color: 'text.secondary' },
        '& .textPrimary': { color: 'text.primary' }
      }}
    >
      <DataGrid
        autoHeight
        rows={rows}
        columns={columns}
        editMode="row"
        hideFooter
        disableRowSelectionOnClick
        rowModesModel={rowModesModel}
        onRowModesModelChange={handleRowModesModelChange}
        processRowUpdate={processRowUpdate}
        onProcessRowUpdateError={handleProcessRowUpdateError}
        slots={{ toolbar: EditToolbar }}
        slotProps={{ toolbar: { setRows, setRowModesModel } }}
      />
    </Box>
  );
};

RelayEndpointsDataGrid.propTypes = {
  endpoints: PropTypes.string,
  onChange: PropTypes.func.isRequired
};

export default RelayEndpointsDataGrid;
