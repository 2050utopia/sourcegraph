import { AdjustmentDirection, DOMFunctions, PositionAdjuster } from '@sourcegraph/codeintellify'
import { of } from 'rxjs'
import { FileSpec, RepoSpec, ResolvedRevSpec, RevSpec } from '../../../../../shared/src/util/url'
import { querySelectorOrSelf } from '../../shared/util/dom'
import { CodeHost, CodeViewSpecResolver, CodeViewSpecWithOutSelector, MountGetter } from '../code_intelligence'
import { getContext } from './context'
import { diffDOMFunctions, singleFileDOMFunctions } from './dom_functions'
import {
    resolveCommitViewFileInfo,
    resolveCompareFileInfo,
    resolveFileInfoForSingleFileSourceView,
    resolvePullRequestFileInfo,
    resolveSingleFileDiffFileInfo,
} from './file_info'
import { isCommitsView, isCompareView, isPullRequestView, isSingleFileView } from './scrape'

const createToolbarMount = (codeView: HTMLElement) => {
    const existingMount = codeView.querySelector<HTMLElement>('.sg-toolbar-mount')
    if (existingMount) {
        return existingMount
    }

    const fileActions = codeView.querySelector<HTMLElement>('.file-toolbar .secondary')
    if (!fileActions) {
        throw new Error('Unable to find mount location')
    }

    fileActions.style.display = 'flex'

    const mount = document.createElement('div')
    mount.classList.add('btn-group')
    mount.classList.add('sg-toolbar-mount')
    mount.classList.add('sg-toolbar-mount-bitbucket-server')

    fileActions.insertAdjacentElement('afterbegin', mount)

    return mount
}

/**
 * Sometimes tabs are converted to spaces so we need to adjust. Luckily, there
 * is an attribute `cm-text` that contains the real text.
 */
const createPositionAdjuster = (
    dom: DOMFunctions
): PositionAdjuster<RepoSpec & RevSpec & FileSpec & ResolvedRevSpec> => ({ direction, codeView, position }) => {
    const codeElement = dom.getCodeElementFromLineNumber(codeView, position.line, position.part)
    if (!codeElement) {
        throw new Error('(adjustPosition) could not find code element for line provided')
    }

    let delta = 0
    for (const modifiedTextElem of codeElement.querySelectorAll('[cm-text]')) {
        const actualText = modifiedTextElem.getAttribute('cm-text') || ''
        const adjustedText = modifiedTextElem.textContent || ''

        delta += actualText.length - adjustedText.length
    }

    const modifier = direction === AdjustmentDirection.ActualToCodeView ? -1 : 1

    const newPos = {
        line: position.line,
        character: position.character + modifier * delta,
    }

    return of(newPos)
}

const toolbarButtonProps = {
    className: 'aui-button',
    style: { marginLeft: 10 },
}

/**
 * A code view spec for single file code view in the "source" view (not diff).
 */
const singleFileSourceCodeView: CodeViewSpecWithOutSelector = {
    getToolbarMount: createToolbarMount,
    dom: singleFileDOMFunctions,
    resolveFileInfo: resolveFileInfoForSingleFileSourceView,
    adjustPosition: createPositionAdjuster(singleFileDOMFunctions),
    toolbarButtonProps,
}

const baseDiffCodeView = {
    getToolbarMount: createToolbarMount,
    dom: diffDOMFunctions,
    adjustPosition: createPositionAdjuster(diffDOMFunctions),
    toolbarButtonProps,
}

/**
 * A code view spec for a single file "diff to previous" view
 */
const singleFileDiffCodeView: CodeViewSpecWithOutSelector = {
    ...baseDiffCodeView,
    resolveFileInfo: resolveSingleFileDiffFileInfo,
}

/**
 * A code view spec for pull requests
 */
const pullRequestDiffCodeView: CodeViewSpecWithOutSelector = {
    ...baseDiffCodeView,
    resolveFileInfo: resolvePullRequestFileInfo,
}

/**
 * A code view spec for compare pages
 */
const compareDiffCodeView: CodeViewSpecWithOutSelector = {
    ...baseDiffCodeView,
    resolveFileInfo: resolveCompareFileInfo,
}

/**
 * A code view spec for commit pages
 */
const commitDiffCodeView: CodeViewSpecWithOutSelector = {
    ...baseDiffCodeView,
    resolveFileInfo: resolveCommitViewFileInfo,
}

const codeViewSpecResolver: CodeViewSpecResolver = {
    selector: '.file-content',
    resolveCodeViewSpec: codeView => {
        const contentView = codeView.querySelector('.content-view')
        if (!contentView) {
            return null
        }
        if (isCompareView()) {
            return compareDiffCodeView
        }
        if (isCommitsView()) {
            return commitDiffCodeView
        }
        if (isSingleFileView(codeView)) {
            const isDiff = contentView.classList.contains('diff-view')
            return isDiff ? singleFileDiffCodeView : singleFileSourceCodeView
        }
        if (isPullRequestView()) {
            return pullRequestDiffCodeView
        }
        console.error('Unknown code view', codeView)
        return null
    },
}

const getCommandPaletteMount: MountGetter = (container: HTMLElement): HTMLElement | null => {
    const headerElement = querySelectorOrSelf(container, '.aui-header-primary .aui-nav')
    if (!headerElement) {
        return null
    }
    const classes = ['command-palette-button', 'command-palette-button__bitbucket-server']
    const create = (): HTMLElement => {
        const mount = document.createElement('li')
        mount.className = classes.join(' ')
        headerElement.insertAdjacentElement('beforeend', mount)
        return mount
    }
    const preexisting = headerElement.querySelector<HTMLElement>(classes.map(c => `.${c}`).join(''))
    return preexisting || create()
}

function getViewContextOnSourcegraphMount(container: HTMLElement): HTMLElement | null {
    const branchSelectorButtons = querySelectorOrSelf(container, '.branch-selector-toolbar .aui-buttons')
    if (!branchSelectorButtons) {
        return null
    }
    const preexisting = branchSelectorButtons.querySelector<HTMLElement>('#open-on-sourcegraph')
    if (preexisting) {
        return preexisting
    }
    const mount = document.createElement('span')
    mount.id = 'open-on-sourcegraph'
    mount.className = 'open-on-sourcegraph--bitbucket-server'
    branchSelectorButtons.insertAdjacentElement('beforeend', mount)
    return mount
}

export const bitbucketServerCodeHost: CodeHost = {
    name: 'bitbucket-server',
    check: () =>
        !!document.querySelector('.bitbucket-header-logo') ||
        !!document.querySelector('.aui-header-logo.aui-header-logo-bitbucket'),
    codeViewSpecResolver,
    getCommandPaletteMount,
    commandPalettePopoverClassName: 'command-palette-popover--bitbucket-server',
    actionNavItemClassProps: {
        actionItemClass: 'aui-button action-item__bitbucket-server',
        listItemClass: 'aui-buttons',
    },
    codeViewToolbarClassName: 'code-view-toolbar--bitbucket-server',
    getViewContextOnSourcegraphMount,
    getContext,
    contextButtonClassName: 'aui-button',
}
