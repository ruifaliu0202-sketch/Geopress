import { useState } from 'react';
import type { ReactNode } from 'react';
import {
  AppBar,
  Box,
  Button,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Stack,
  Toolbar,
  Tooltip,
  Typography,
} from '@mui/material';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import MenuIcon from '@mui/icons-material/Menu';
import type { NavItem, ViewKey } from '../../appTypes';

const expandedNavWidth = 224;
const collapsedNavWidth = 72;
const topBarHeight = 64;

function BrandMark({ collapsed }: { collapsed?: boolean }) {
  return (
    <Stack
      direction="row"
      alignItems="center"
      spacing={1.25}
      sx={{
        minWidth: 0,
        justifyContent: collapsed ? 'center' : 'flex-start',
        width: '100%',
      }}
    >
      <Box
        sx={{
          display: 'grid',
          placeItems: 'center',
          width: 34,
          height: 34,
          flex: '0 0 auto',
          borderRadius: 1,
          bgcolor: 'primary.main',
          color: 'primary.contrastText',
          fontWeight: 800,
        }}
      >
        G
      </Box>
      {!collapsed && (
        <Box sx={{ minWidth: 0 }}>
          <Typography variant="h3" sx={{ lineHeight: 1 }}>
            Geopress
          </Typography>
          <Typography variant="caption" color="text.secondary" noWrap>
            内容自动发布平台
          </Typography>
        </Box>
      )}
    </Stack>
  );
}

function NavigationList({
  activeView,
  items,
  collapsed,
  mobile,
  onNavigate,
}: {
  activeView: ViewKey;
  items: NavItem[];
  collapsed: boolean;
  mobile?: boolean;
  onNavigate: (view: ViewKey) => void;
}) {
  return (
    <List disablePadding sx={{ display: 'grid', gap: 0.5 }}>
      {items.map((item) => {
        const selected = activeView === item.key;
        const button = (
          <ListItemButton
            key={item.key}
            selected={selected}
            onClick={() => onNavigate(item.key)}
            data-tour-id={`${mobile ? 'mobile-nav' : 'nav'}-${item.key}`}
            sx={{
              minHeight: 42,
              justifyContent: collapsed ? 'center' : 'flex-start',
              px: collapsed ? 1.25 : 1.5,
              borderRadius: 1,
              '&.Mui-selected': {
                bgcolor: 'primary.main',
                color: 'primary.contrastText',
                '&:hover': {
                  bgcolor: 'primary.dark',
                },
                '& .MuiListItemIcon-root': {
                  color: 'inherit',
                },
              },
            }}
          >
            <ListItemIcon
              sx={{
                minWidth: collapsed ? 0 : 36,
                color: selected ? 'inherit' : 'text.secondary',
                justifyContent: 'center',
              }}
            >
              {item.icon}
            </ListItemIcon>
            {!collapsed && <ListItemText primary={item.label} primaryTypographyProps={{ fontWeight: 700 }} />}
          </ListItemButton>
        );

        if (!collapsed) {
          return button;
        }

        return (
          <Tooltip key={item.key} title={item.label} placement="right">
            {button}
          </Tooltip>
        );
      })}
    </List>
  );
}

function DesktopSideNav({
  activeView,
  items,
  collapsed,
  onNavigate,
  onToggle,
}: {
  activeView: ViewKey;
  items: NavItem[];
  collapsed: boolean;
  onNavigate: (view: ViewKey) => void;
  onToggle: () => void;
}) {
  return (
    <Box
      component="aside"
      sx={{
        display: { xs: 'none', lg: 'flex' },
        position: 'sticky',
        top: 0,
        height: '100vh',
        width: collapsed ? collapsedNavWidth : expandedNavWidth,
        flex: `0 0 ${collapsed ? collapsedNavWidth : expandedNavWidth}px`,
        flexDirection: 'column',
        borderRight: '1px solid',
        borderColor: 'divider',
        bgcolor: 'background.paper',
        transition: (theme) =>
          theme.transitions.create(['width', 'flex-basis'], {
            duration: theme.transitions.duration.shorter,
          }),
      }}
    >
      <Stack spacing={1.5} sx={{ p: 1.5, minHeight: 0, flex: 1 }}>
        <Stack
          direction="row"
          alignItems="center"
          justifyContent={collapsed ? 'center' : 'space-between'}
          spacing={1}
          sx={{ minHeight: 42 }}
        >
          <BrandMark collapsed={collapsed} />
          {!collapsed && (
            <Tooltip title="折叠菜单">
              <IconButton size="small" onClick={onToggle} aria-label="折叠菜单">
                <ChevronLeftIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
        </Stack>
        {collapsed && (
          <Tooltip title="展开菜单" placement="right">
            <IconButton size="small" onClick={onToggle} aria-label="展开菜单" sx={{ alignSelf: 'center' }}>
              <ChevronRightIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        )}
        <Divider />
        <Box sx={{ overflowY: 'auto', pr: collapsed ? 0 : 0.5 }}>
          <NavigationList activeView={activeView} items={items} collapsed={collapsed} onNavigate={onNavigate} />
        </Box>
      </Stack>
    </Box>
  );
}

function MobileTopNav({
  activeView,
  items,
  onNavigate,
}: {
  activeView: ViewKey;
  items: NavItem[];
  onNavigate: (view: ViewKey) => void;
}) {
  const [open, setOpen] = useState(false);
  const navigate = (view: ViewKey) => {
    onNavigate(view);
    setOpen(false);
  };

  return (
    <>
      <Button
        startIcon={<MenuIcon />}
        variant="outlined"
        onClick={() => setOpen(true)}
        data-tour-id="mobile-nav-menu"
        sx={{ display: { xs: 'inline-flex', lg: 'none' }, flex: '0 0 auto' }}
      >
        菜单
      </Button>
      <Drawer
        open={open}
        onClose={() => setOpen(false)}
        ModalProps={{ keepMounted: true }}
        PaperProps={{
          sx: {
            width: 288,
            maxWidth: 'calc(100vw - 32px)',
            p: 2,
          },
        }}
      >
        <Stack spacing={2} sx={{ minHeight: '100%' }}>
          <BrandMark />
          <Divider />
          <NavigationList activeView={activeView} items={items} collapsed={false} mobile onNavigate={navigate} />
        </Stack>
      </Drawer>
    </>
  );
}

export function WorkspaceShell({
  activeView,
  navItems,
  topShortcuts,
  rightContext,
  children,
  onNavigate,
}: {
  activeView: ViewKey;
  navItems: NavItem[];
  topShortcuts: ReactNode;
  rightContext?: ReactNode;
  children: ReactNode;
  onNavigate: (view: ViewKey) => void;
}) {
  const [navCollapsed, setNavCollapsed] = useState(false);

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default', display: 'flex' }}>
      <DesktopSideNav
        activeView={activeView}
        items={navItems}
        collapsed={navCollapsed}
        onNavigate={onNavigate}
        onToggle={() => setNavCollapsed((value) => !value)}
      />

      <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        <AppBar
          position="sticky"
          color="inherit"
          elevation={0}
          sx={{
            top: 0,
            borderBottom: '1px solid',
            borderColor: 'divider',
            bgcolor: 'background.paper',
            zIndex: (theme) => theme.zIndex.appBar,
          }}
        >
          <Toolbar
            sx={{
              minHeight: topBarHeight,
              gap: 1.5,
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <Stack direction="row" alignItems="center" spacing={1.25} sx={{ minWidth: 0, flex: { xs: 1, lg: '0 0 auto' } }}>
              <MobileTopNav activeView={activeView} items={navItems} onNavigate={onNavigate} />
              <Box sx={{ display: { xs: 'none', lg: 'block' } }}>
                <Typography variant="h3">工作区</Typography>
              </Box>
            </Stack>
            <Box
              sx={{
                minWidth: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'flex-end',
                gap: 1,
                flexWrap: 'wrap',
              }}
            >
              {topShortcuts}
            </Box>
          </Toolbar>
        </AppBar>

        <Box
          sx={{
            flex: 1,
            minHeight: 0,
            p: { xs: 2, md: 3 },
          }}
        >
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: {
                xs: 'minmax(0, 1fr)',
                lg: rightContext ? 'minmax(0, 1fr) 320px' : 'minmax(0, 1fr)',
              },
              gap: 3,
              alignItems: 'start',
              maxWidth: 1680,
              mx: 'auto',
            }}
          >
            <Box component="main" sx={{ minWidth: 0 }}>
              {children}
            </Box>
            {rightContext && (
              <Box
                component="aside"
                sx={{
                  minWidth: 0,
                  display: 'block',
                }}
              >
                {rightContext}
              </Box>
            )}
          </Box>
        </Box>
      </Box>
    </Box>
  );
}
