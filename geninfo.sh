#!/usr/bin/env bash
#
# Author: Ryan Qian <i@bitbili.net>
#

WORKDIR=$(dirname $(realpath ${0}))
. "${WORKDIR}/modules-list"

github_graphql="https://api.github.com/graphql"
github_api="https://api.github.com"
main_repo_api="${github_api}/repos/${GITHUB_REPO:-bekcpear/gopkg-vendors}"

declare -a modules sources paths subpaths skipvers
declare -A index
declare -i i=0

while read -r line; do
  if [[ ${line} =~ ^[[:space:]]*$ ]]; then
    i=$((i + 1))
    continue
  fi
  read _k _v <<<"${line}"
  case ${_k} in
    module:)
      eval "modules[${i}]='${_v}'"
      eval "index[${_v}]='${i}'"
      ;;
    source:)
      eval "sources[${i}]='${_v}'"
      ;;
    path:)
      eval "paths[${i}]='${_v}'"
      ;;
    subpath:)
      eval "subpaths[${i}]='${_v}'"
      ;;
    skipver:)
      eval "skipvers[${i}]='${_v}'"
      ;;
    *)
      echo "error: invalid key '${_k}'" >&2
      ;;
  esac
done <<<"${LIST}"
unset _k _v

while :; do
  case ${1} in
  --matrix-list)
    MODE="matrix-list"
    shift
    break
    ;;
  --existing-branch)
    MODE="existing-branch"
    shift
    BRANCH="${1}"
    shift
    break
    ;;
  --has-an-update)
    MODE="has-an-update"
    shift
    MODULE="${1}"
    shift
    break
    ;;
  --download-latest)
    MODE="download-latest"
    shift
    MODULE="${1}"
    TAG="${2}"
    DST="${3}"
    shift 3
    break
    ;;
  --get-subpath)
    MODE="get-subpath"
    shift
    MODULE="${1}"
    shift
    break
  esac
done

# Echo the first tag name from a GitHub GraphQL 'refs' response (already
# ordered by TAG_COMMIT_DATE DESC) that does not match the ERE in ${2}.
# When a prefix is given in ${3}, only names starting with it qualify and the
# prefix is stripped from the result. Skipped names are reported on stderr so
# they never end up in the captured stdout. Returns 1 when nothing qualifies.
__gg_pick_latest_tag() {
  local res=${1} skip=${2} prefix=${3} name
  while [[ ${res} =~ \"name\":\"([^\"]+)\" ]]; do
    name=${BASH_REMATCH[1]}
    res=${res#*"${BASH_REMATCH[0]}"}
    if [[ -n ${prefix} ]]; then
      [[ ${name} == ${prefix}* ]] || continue
      name=${name#"${prefix}"}
    fi
    if [[ -n ${skip} && ${name} =~ ${skip} ]]; then
      echo "skip version: ${prefix}${name}" >&2
      continue
    fi
    echo -n "${name}"
    return 0
  done
  return 1
}

case ${MODE} in
  matrix-list)
    for _m in "${modules[@]}"; do
      _module+="\"${_m}\", "
    done
    set -x
    echo "list={\"module\": [${_module%, }]}" >>$GITHUB_OUTPUT
    ;;
  existing-branch)
    _rc=$(curl -w '%{http_code}' -o /dev/null \
      -H "Accept: application/vnd.github+json" \
      -H "Authorization: Bearer $GITHUB_TOKEN" \
      -H "X-GitHub-Api-Version: 2022-11-28" \
      "${main_repo_api}/branches/${BRANCH}")
    set -x
    if [[ ${_rc} == 200 ]]; then
      echo "exists=true" >>$GITHUB_OUTPUT
    else
      echo "exists=false" >>$GITHUB_OUTPUT
    fi
    ;;
  has-an-update)
    . "${WORKDIR}/.funcs/_vercmp.sh"
    i=${index[${MODULE}]}
    TYPE=${sources[$i]}
    RPATH=${paths[$i]}
    _SKIP=${skipvers[$i]}
    _current_tag_prefix="vendor-${MODULE}-"
    _gg_query_tag_f="{
        \"query\": \"query { \
          repository(owner: \\\\\\\"%s\\\\\\\", name: \\\\\\\"%s\\\\\\\") { \
            refs(refPrefix: \\\\\\\"refs/tags/\\\\\\\", %s orderBy: \
              { field: TAG_COMMIT_DATE, direction: DESC }, first: 10) { \
              edges { \
                node { \
                  name \
                } \
              } \
            } \
          } \
        }\"
      }"
    printf -v _gg_query_tag "${_gg_query_tag_f}" ${GITHUB_REPO%/*} ${GITHUB_REPO#*/} \
      "query: \\\"${_current_tag_prefix}\\\", "
    _res=$(curl -X POST \
      -H "Accept: application/json" \
      -H "Authorization: Bearer $GITHUB_TOKEN" \
      -d "${_gg_query_tag}" ${github_graphql} )
    _current=$(__gg_pick_latest_tag "${_res}" "${_SKIP}" "${_current_tag_prefix}")
    echo "current version: ${_current}"
    case ${TYPE} in
      github)
        printf -v _gg_query_tag "${_gg_query_tag_f}" ${RPATH%/*} ${RPATH#*/}
        _res=$(curl -X POST \
          -H "Accept: application/json" \
          -H "Authorization: Bearer $GITHUB_TOKEN" \
          -d "${_gg_query_tag}" ${github_graphql} )
        _latest=$(__gg_pick_latest_tag "${_res}" "${_SKIP}")
        echo "latest version: ${_latest}"
        ;;
    esac
    if [[ -n ${_latest} ]] && _vercmp g ${_latest} ${_current:-0}; then
      echo "declare -A tag_name[${MODULE}]='${_latest}'" >>./tag_name
    fi
    ;;
  download-latest)
    i=${index[${MODULE}]}
    TYPE=${sources[$i]}
    RPATH=${paths[$i]}
    case $TYPE in
      github)
        URL="https://github.com/$RPATH/archive/refs/tags/$TAG.tar.gz"
    esac
    curl -fL -o $DST $URL
    ;;
  get-subpath)
    i=${index[${MODULE}]}
    echo -n ${subpaths[$i]}
    ;;
esac
