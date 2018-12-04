import marked from 'marked'
import FileIcon from 'mdi-react/FileIcon'
import React from 'react'
import * as GQL from '../../../shared/src/graphql/schema'
import { ResultContainer } from './ResultContainer'
import { SearchResultMatch } from './SearchResultMatch'

export interface HighlightRange {
    /**
     * The 0-based line number that this highlight appears in
     */
    line: number
    /**
     * The 0-based character offset to start highlighting at
     */
    character: number
    /**
     * The number of characters to highlight
     */
    length: number
}

interface Props {
    result: GQL.GenericSearchResult
    isLightTheme: boolean
}

export class SearchResult extends React.Component<Props> {
    constructor(props: Props) {
        super(props)
    }

    private renderTitle = () => (
        <div className="repository-search-result__title">
            <span
                dangerouslySetInnerHTML={{
                    __html: marked(this.props.result.label, { gfm: true, breaks: true, sanitize: true }),
                }}
            />
            {this.props.result.detail && (
                <>
                    <span className="repository-search-result__spacer" />
                    <small
                        dangerouslySetInnerHTML={{
                            __html: marked(this.props.result.detail, { gfm: true, breaks: true, sanitize: true }),
                        }}
                    />
                </>
            )}
        </div>
    )

    private renderBody = () => (
        <>
            {this.props.result.matches.map(item => {
                const highlightRanges: HighlightRange[] = []
                item.highlights.map(highlight =>
                    highlightRanges.push({
                        line: highlight.line,
                        character: highlight.character,
                        length: highlight.length,
                    })
                )

                return (
                    <SearchResultMatch
                        key={item.url}
                        item={item}
                        body={item.body}
                        url={item.url}
                        highlightRanges={highlightRanges}
                        isLightTheme={this.props.isLightTheme}
                    />
                )
            })}
        </>
    )

    public render(): JSX.Element {
        return (
            <ResultContainer
                stringIcon={this.props.result.icon}
                icon={FileIcon}
                collapsible={this.props.result && this.props.result.matches.length > 0}
                defaultExpanded={true}
                title={this.renderTitle()}
                expandedChildren={this.renderBody()}
            />
        )
    }
}
