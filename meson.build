project('installer',
  version: '0.1.0',
  meson_version: '>= 1.0.0',
  default_options: [
    'warning_level=2',
    'werror=false',
  ],
)

i18n = import('i18n')

go_root = run_command('printenv', 'GOROOT', check: false)

if go_root.returncode() != 0
  go_bin = find_program('go')
else
  go_bin = find_program(join_paths(go_root.stdout().strip(), 'bin', 'go'))
endif

LOCALES_DIR = get_option('prefix') / get_option('localedir')
LOG_FILEPATH = get_option('localstatedir') / meson.project_name() / meson.project_name() + '.log'

constants = [
  ['installer/lib.BuildPathLocales', LOCALES_DIR],
  ['installer/lib.BuildPathLogFile', LOG_FILEPATH],
]

ldflags = []
foreach constant : constants
  ldflags += '-X \'@0@=@1@\''.format(constant[0], constant[1])
endforeach

subdir('po')

custom_target(
  'go-build',
  build_by_default: true,
  build_always_stale: true,
  output: meson.project_name(),
  console: true,
  install: true,
  install_dir: get_option('bindir'),
  command: [go_bin, 'build', '-v', '-ldflags', ' '.join(ldflags), '-o', '@OUTPUT@', meson.current_source_dir()]
)
